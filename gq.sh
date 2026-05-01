#!/usr/bin/env bash

# gq — git quick push  (one commit per file)

# ── colours ─────────────────────────────────
R='\033[0;31m'
G='\033[0;32m'
Y='\033[0;33m'
M='\033[0;35m'
C='\033[0;36m'
W='\033[1;37m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

TAG_FEAT="\033[1;32m[feat]\033[0m"
TAG_FIX="\033[1;31m[fix]\033[0m"
TAG_DOCS="\033[1;34m[docs]\033[0m"
TAG_REFACTOR="\033[1;36m[refactor]\033[0m"
TAG_STYLE="\033[1;35m[style]\033[0m"
TAG_PERF="\033[1;33m[perf]\033[0m"
TAG_TEST="\033[1;37m[test]\033[0m"
TAG_CHORE="\033[2;37m[chore]\033[0m"
TAG_DEPLOY="\033[1;34m[deploy]\033[0m"
TAG_REMOVE="\033[1;31m[remove]\033[0m"
TAG_SECURITY="\033[1;33m[security]\033[0m"
TAG_REVERT="\033[2;33m[revert]\033[0m"

PRESET_TAGS=(
  "$TAG_FEAT" "$TAG_FIX" "$TAG_DOCS" "$TAG_REFACTOR"
  "$TAG_STYLE" "$TAG_PERF" "$TAG_TEST" "$TAG_CHORE"
  "$TAG_DEPLOY" "$TAG_REMOVE" "$TAG_SECURITY" "$TAG_REVERT"
)
PRESET_PREFIXES=(
  "feat: " "fix: " "docs: " "refactor: "
  "style: " "perf: " "test: " "chore: "
  "deploy: " "remove: " "security: " "revert: "
)

# ── helpers ──────────────────────────────────
banner() {
  echo -e "\n${DIM}────────────────────────────────────────${NC}"
  echo -e "  ${W}${BOLD}gq${NC}  ${DIM}git quick push${NC}"
  echo -e "${DIM}────────────────────────────────────────${NC}\n"
}
ok()      { echo -e "  ${G}+${NC}  $*"; }
info()    { echo -e "  ${C}:${NC}  $*"; }
warn()    { echo -e "  ${Y}!${NC}  $*"; }
err()     { echo -e "  ${R}x${NC}  $*"; }
divider() { echo -e "  ${DIM}──────────────────────────────────────${NC}"; }

# ── git check ───────────────────────────────
if ! git rev-parse --git-dir &>/dev/null; then
  banner; err "Not inside a git repository."; exit 1
fi

BRANCH=$(git symbolic-ref --short HEAD 2>/dev/null || echo "detached")
REMOTE=$(git remote | head -1)
REMOTE="${REMOTE:-origin}"

# ── push ─────────────────────────────────────
do_git_push() {
  info "pushing  ${DIM}-> $BRANCH${NC}"
  if git push --quiet 2>/dev/null; then
    ok "pushed"
  else
    warn "no upstream — setting now..."
    if git push --set-upstream "$REMOTE" "$BRANCH"; then
      ok "pushed  ${DIM}(upstream: $REMOTE/$BRANCH)${NC}"
    else
      err "push failed. check remote / credentials."
      exit 1
    fi
  fi
}

# ── flags ────────────────────────────────────
AUTO_YES=false
GLOBAL_MSG=""

while [ $# -gt 0 ]; do
  case "$1" in
    -m|--message) GLOBAL_MSG="$2"; shift 2 ;;
    -y|--yes)     AUTO_YES=true; shift ;;
    -h|--help)
      banner
      printf "  usage:\n"
      printf "    gq                interactive — per file\n"
      printf "    gq -m 'msg'       same message for all files\n"
      printf "    gq -y -m 'msg'    no prompts at all\n\n"
      exit 0 ;;
    *) shift ;;
  esac
done

# ── main ─────────────────────────────────────
banner

# collect files into indexed arrays — NO while-read loop, avoids stdin clash
FILE_COUNT=0
i=0
while IFS= read -r f; do
  [ -z "$f" ] && continue
  eval "FILE_$i=\"\$f\""
  i=$((i+1))
done < <(
  { git diff --name-only; git ls-files --others --exclude-standard; git diff --cached --name-only; } | sort -u
)
FILE_COUNT=$i

if [ "$FILE_COUNT" -eq 0 ]; then
  warn "Nothing to commit. Working tree clean."
  exit 0
fi

echo -e "  ${DIM}branch${NC}  ${M}${BOLD}$BRANCH${NC}   ${DIM}files${NC}  ${W}$FILE_COUNT${NC}\n"

COMMITTED=0
STOP=false

i=0
while [ $i -lt "$FILE_COUNT" ]; do
  eval "filepath=\$FILE_$i"
  i=$((i+1))

  echo ""
  divider
  echo -e "  ${W}${BOLD}$filepath${NC}"
  divider

  FULL_MSG=""

  if [ -n "$GLOBAL_MSG" ]; then
    if [ "$AUTO_YES" = false ]; then
      printf "  ${DIM}\"%s\"${NC}  commit? ${W}[Y/n/q]${NC} " "$GLOBAL_MSG"
      read -r c < /dev/tty
      case "$c" in
        [Qq]*) STOP=true; break ;;
        [Nn]*) warn "skipped"; echo ""; continue ;;
      esac
    fi
    FULL_MSG="$GLOBAL_MSG"
  else
    j=1
    for tag in "${PRESET_TAGS[@]}"; do
      printf "  ${C}%2d${NC}  %b\n" "$j" "$tag"
      j=$((j+1))
    done
    echo -e "  ${C} c${NC}  ${DIM}custom${NC}   ${C}s${NC}  ${DIM}skip${NC}   ${C}q${NC}  ${DIM}quit${NC}"
    echo ""
    printf "  ${W}> ${NC}"
    read -r choice < /dev/tty

    case "$choice" in
      q) STOP=true; break ;;
      s) warn "skipped"; echo ""; continue ;;
      c)
        printf "  ${W}message: ${NC}"
        read -r FULL_MSG < /dev/tty
        [ -z "$FULL_MSG" ] && { warn "empty — skipped"; continue; }
        ;;
      *)
        if echo "$choice" | grep -qE '^[0-9]+$' && [ "$choice" -ge 1 ] && [ "$choice" -le "${#PRESET_PREFIXES[@]}" ]; then
          idx=$((choice-1))
          tag="${PRESET_TAGS[$idx]}"
          prefix="${PRESET_PREFIXES[$idx]}"
          printf "  %b ${DIM}+${NC} " "$tag"
          read -r suffix < /dev/tty
          [ -z "$suffix" ] && { warn "empty — skipped"; continue; }
          FULL_MSG="${prefix}${suffix}"
        else
          warn "invalid — skipped"
          continue
        fi
        ;;
    esac
  fi

  git add -- "$filepath"

  if git diff --cached --quiet; then
    warn "nothing to stage for $filepath"
    continue
  fi

  if git commit -m "$FULL_MSG" --quiet; then
    ok "${G}${BOLD}$FULL_MSG${NC}"
    COMMITTED=$((COMMITTED+1))
  else
    err "commit failed — $filepath"
  fi

done

echo ""

if [ "$COMMITTED" -eq 0 ]; then
  warn "No commits made."
  exit 0
fi

divider
do_git_push
divider
echo -e "\n  ${G}${BOLD}done.${NC}  ${DIM}$COMMITTED commit(s) pushed${NC}\n"
