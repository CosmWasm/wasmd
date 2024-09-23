is_not_equal () {
  if [ "$1" == "$2" ]; then echo "0"; else echo "1"; fi
}
retry_exec () {
  local check
  for (( ; ; ))
  do
    check=$(eval $1)
    local not_equal=$(is_not_equal "$check" $2)
    if [ "$not_equal" == "0" ]; then sleep 1; else break; fi
    # if [ "$check" == "null" ]; then sleep 1; else break; fi
  done

  # echo "check: $check"
  echo $check
}
