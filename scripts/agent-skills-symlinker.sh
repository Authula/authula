#!/bin/bash

# Symlinks the skills from .agents/skills to other agent directories

rm -rf .claude/skills .qwen/skills

mkdir -p .claude .qwen
for dir in .claude .qwen; do
  ln -sf ../.agents/skills "$dir/skills"
done
