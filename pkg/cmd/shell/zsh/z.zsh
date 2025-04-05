export Z_SOURCED=1

# wrapper for z command
z() {
  local output
  local exit_code

  if [[ "$1" == "project" && "$2" == "select" ]]; then
    output="$(command z "$@")"
    exit_code=$?

    if [[ $exit_code -ne 0 ]]; then
      echo "Error: 'z project select' failed with exit code $exit_code" >&2
      echo "$output" >&2
      return $exit_code
    fi

    if [[ -d "${output}" ]]; then
      cd "${output}"
      return 0
    else
      echo "${output}"
      return 1
    fi

  else
    command z "$@"
    return $?
  fi
}
