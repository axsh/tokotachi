#!/bin/bash
set -e

# scripts/utils/migration/restruct_prompts_folder_to_v20260604.sh

SRC_DIR="prompts_old/phases/000-foundation"
DST_DIR="prompts/phases/000-foundation"
ADAPT_MODE=false

show_help() {
  echo "Usage: $0 [options]"
  echo ""
  echo "Options:"
  echo "  -h, --help    Show this help message"
  echo "  --adapt       Execute the actual migration (moves files and deletes old folders)"
  echo ""
  echo "By default, this script runs in dry-run mode and will not modify any files."
}

# Parse options
while [[ "$#" -gt 0 ]]; do
  case $1 in
    -h|--help)
      show_help
      exit 0
      ;;
    --adapt)
      ADAPT_MODE=true
      shift
      ;;
    *)
      echo "Error: Unknown option $1" >&2
      show_help >&2
      exit 1
      ;;
  esac
done

if [ ! -d "${SRC_DIR}/ideas" ]; then
  echo "Error: Source ideas directory does not exist: ${SRC_DIR}/ideas" >&2
  exit 1
fi

if [ "${ADAPT_MODE}" = false ]; then
  echo "========================================="
  echo " Running in DRY-RUN mode"
  echo "========================================="
else
  echo "========================================="
  echo " Running in ADAPT (migration) mode"
  echo "========================================="
fi

# Find all branch directories in the old ideas folder
# Ignore .gitkeep and other non-directories
find "${SRC_DIR}/ideas" -mindepth 1 -maxdepth 1 -type d | sort | while read -r branch_dir; do
  branch_name=$(basename "${branch_dir}")
  
  old_ideas_dir="${SRC_DIR}/ideas/${branch_name}"
  new_ideas_dir="${DST_DIR}/branches/${branch_name}/ideas"
  
  old_plans_dir="${SRC_DIR}/plans/${branch_name}"
  new_plans_dir="${DST_DIR}/branches/${branch_name}/plans"
  
  if [ "${ADAPT_MODE}" = true ]; then
    echo "Processing branch: ${branch_name}"
    
    # Migrate ideas
    if [ -d "${old_ideas_dir}" ]; then
      mkdir -p "${new_ideas_dir}"
      if [ "$(ls -A "${old_ideas_dir}")" ]; then
        # Use cp & rm style or mv to make sure it doesn't fail across filesystems,
        # but inside same workspace mv is fine.
        mv "${old_ideas_dir}"/* "${new_ideas_dir}/"
        echo "  [OK] Moved ideas to ${new_ideas_dir}"
      else
        echo "  [Info] Old ideas folder was empty"
      fi
    fi
    
    # Migrate plans
    if [ -d "${old_plans_dir}" ]; then
      mkdir -p "${new_plans_dir}"
      if [ "$(ls -A "${old_plans_dir}")" ]; then
        mv "${old_plans_dir}"/* "${new_plans_dir}/"
        echo "  [OK] Moved plans to ${new_plans_dir}"
      else
        echo "  [Info] Old plans folder was empty"
      fi
    fi
    
    # Cleanup old directories
    rm -rf "${old_ideas_dir}"
    rm -rf "${old_plans_dir}"
    echo "  [OK] Cleaned up old directories: ${old_ideas_dir}, ${old_plans_dir}"
    
  else
    # Dry-run Mode
    echo "Branch: ${branch_name}"
    
    if [ -d "${old_ideas_dir}" ] && [ "$(ls -A "${old_ideas_dir}")" ]; then
      echo "  Would move files from ${old_ideas_dir} to ${new_ideas_dir}:"
      ls -A "${old_ideas_dir}" | sed 's/^/    - /'
    else
      echo "  No ideas files to move."
    fi
    
    if [ -d "${old_plans_dir}" ] && [ "$(ls -A "${old_plans_dir}")" ]; then
      echo "  Would move files from ${old_plans_dir} to ${new_plans_dir}:"
      ls -A "${old_plans_dir}" | sed 's/^/    - /'
    else
      echo "  No plans files to move."
    fi
    
    echo "  Would delete directories: ${old_ideas_dir}, ${old_plans_dir}"
  fi
done

if [ "${ADAPT_MODE}" = false ]; then
  echo ""
  echo "Dry-run finished. To apply these changes, run the script with the --adapt option:"
  echo "  $0 --adapt"
fi
