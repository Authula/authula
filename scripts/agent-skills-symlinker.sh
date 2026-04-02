#!/bin/bash

# Symlinks the skills from .agents/skills to other agent directories

mkdir -p .github .claude .gemini .qwen .opencode
for dir in .github .claude .gemini .qwen .opencode; do
  ln -sf ../.agents/skills "$dir/skills"
done
