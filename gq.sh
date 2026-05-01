#!/usr/bin/env bash

# gq — git quick push

# ── colours ─────────────────────────────────
R='\033[0;31m'
G='\033[0;32m'
Y='\033[0;33m'
B='\033[0;34m'
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

# ── helpers ──────────────────────────────────
banner() {
  echo -e "\n${DIM}────────────────────────────────────────${NC}"
  echo -e "  ${W}${BOLD}gq${NC}  ${DIM}git quick push${NC}"
  echo -e "${DIM}────────────────────────────────────────${NC}\n"
}

ok()   { echo -e "  ${G}+${NC}  $*"; }
info() { echo -e "  ${C}:${NC}  $*"; }
warn() { echo -e "  ${Y}!${NC}  $*"; }
err()  { echo -e "  ${R}x${NC}  $*"; }

divider() { echo -e "  ${DIM}──────────────────────────────────────${NC}"; }

# ── check we're in a git repo ────────────────
if ! git rev-parse --git-dir &>/dev/null; then
  banner
  err "Not inside a git repository."
  exit 1
fi

BRANCH=$(git symbolic-ref --short HEAD 2>/dev/null || echo "detached")

# ── status snapshot ───────────────────────────
show_status() {
  local staged unstaged untracked
  staged=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d ' ')
  unstaged=$(git diff --name-only 2>/dev/null | wc -l | tr -d ' ')
  untracked=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l | tr -d ' ')

  echo -e "  ${DIM}branch${NC}    ${M}${BOLD}$BRANCH${NC}"
  echo -e "  ${DIM}staged${NC}    ${G}$staged${NC}  ${DIM}unstaged${NC} ${Y}$unstaged${NC}  ${DIM}untracked${NC} ${R}$untracked${NC}"

  local total=$(( unstaged + untracked ))
  if [[ $total -gt 0 ]]; then
    echo ""
    git status --short | head -20 | while IFS= read -r line; do
      echo -e "    ${DIM}$line${NC}"
    done
  fi
}

# ── preset list ───────────────────────────────
PRESETS=(
  "$TAG_FEAT|feat: "
  "$TAG_FIX|fix: "
  "$TAG_DOCS|docs: "
  "$TAG_REFACTOR|refactor: "
  "$TAG_STYLE|style: "
  "$TAG_PERF|perf: "
  "$TAG_TEST|test: "
  "$TAG_CHORE|chore: "
  "$TAG_DEPLOY|deploy: "
  "$TAG_REMOVE|remove: "
  "$TAG_SECURITY|security: "
  "$TAG_REVERT|revert: "
)

pick_preset() {
  echo ""
  echo -e "  ${W}${BOLD}Type:${NC}"
  divider
  for i in "${!PRESETS[@]}"; do
    local tag="${PRESETS[$i]%%|*}"
    printf "  ${C}%2d${NC}  %b\n" "$((i+1))" "$tag"
  done
  divider
  echo -e "  ${C} c${NC}  ${DIM}custom${NC}"
  echo -e "  ${C} q${NC}  ${DIM}quit${NC}"
  echo ""
  printf "  ${W}> ${NC}"
  read -r choice

  if [[ "$choice" == "q" ]]; then
    echo ""; info "Aborted."; exit 0
  elif [[ "$choice" == "c" ]]; then
    printf "\n  ${W}message: ${NC}"
    read -r COMMIT_MSG
  elif [[ "$choice" =~ ^[0-9]+$ ]] && (( choice >= 1 && choice <= ${#PRESETS[@]} )); then
    local entry="${PRESETS[$((choice-1))]}"
    local tag="${entry%%|*}"
    local prefix="${entry##*|}"
    printf "\n  %b ${DIM}+${NC} " "$tag"
    read -r suffix
    COMMIT_MSG="${prefix}${suffix}"
  else
    warn "Invalid — using custom."
    printf "\n  ${W}message: ${NC}"
    read -r COMMIT_MSG
  fi

  if [[ -z "$COMMIT_MSG" ]]; then
    err "Empty message. Aborted."
    exit 1
  fi
}

# ── run git commands ──────────────────────────
do_push() {
  echo ""
  divider

  info "staging  ${DIM}git add .${NC}"
  if ! git add . 2>&1; then
    err "git add failed"; exit 1
  fi
  ok "staged"

  info "committing  ${DIM}\"$COMMIT_MSG\"${NC}"
  if ! git commit -m "$COMMIT_MSG" 2>&1; then
    err "commit failed (nothing to commit?)"; exit 1
  fi
  ok "committed"

  info "pushing  ${DIM}-> $BRANCH${NC}"
  if git push 2>&1; then
    ok "pushed"
  else
    warn "no upstream — setting now..."
    REMOTE=$(git remote | head -1)
    if git push --set-upstream "${REMOTE:-origin}" "$BRANCH" 2>&1; then
      ok "pushed  ${DIM}(upstream set: ${REMOTE:-origin}/$BRANCH)${NC}"
    else
      err "push failed. check remote / credentials."
      exit 1
    fi
  fi

  divider
  echo -e "\n  ${G}${BOLD}done.${NC}  ${DIM}$COMMIT_MSG${NC}\n"
}

# ── flags ─────────────────────────────────────
AUTO_YES=false
COMMIT_MSG=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--message) COMMIT_MSG="$2"; shift 2 ;;
    -y|--yes)     AUTO_YES=true; shift ;;
    -h|--help)
      banner
      echo -e "  ${W}usage:${NC}"
      echo -e "    ${C}gq${NC}               interactive"
      echo -e "    ${C}gq -m \"msg\"${NC}      skip menu"
      echo -e "    ${C}gq -y -m \"msg\"${NC}   no prompts, just push\n"
      exit 0 ;;
    *) shift ;;
  esac
done

# ── main ──────────────────────────────────────
banner
show_status
echo ""

if git diff --quiet && git diff --cached --quiet && [[ -z $(git ls-files --others --exclude-standard) ]]; then
  warn "Nothing to commit. Working tree clean."
  exit 0
fi

if [[ -z "$COMMIT_MSG" ]]; then
  pick_preset
fi

if [[ "$AUTO_YES" == false ]]; then
  echo ""
  echo -e "  ${W}message:${NC} ${G}\"$COMMIT_MSG\"${NC}"
  echo -e "  ${W}branch:${NC}  ${M}$BRANCH${NC}"
  echo ""
  printf "  ${W}push? [Y/n]:${NC} "
  read -r confirm
  [[ "$confirm" =~ ^[Nn] ]] && { info "Aborted."; exit 0; }
fi

do_push
