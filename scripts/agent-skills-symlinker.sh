#!/bin/bash

# Symlinks the skills from .agents/skills to other agent directories

rm -rf .github/skills .claude/skills .gemini/skills .qwen/skills .opencode/skills

mkdir -p .github .claude .gemini .qwen .opencode
for dir in .github .claude .gemini .qwen .opencode; do
  ln -sf ../.agents/skills "$dir/skills"
done
