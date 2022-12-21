#!/usr/bin/env sh

regexp='^(?:ci|feat|fix|docs|style|refactor|revert|perf|test|chore)\(?(?:\w+|\s|\-|_)?\)?:\s\w+'

while read line; do
	# ignore merge requests
	if echo "$line" | grep -qE "^Merge branch \'"; then
		continue
	fi

	# check semantic versioning scheme
	if ! echo "$line" | grep -qE $regexp; then
		echo
		echo "Your commit title did not follow semantic versioning: '$line'"
		echo
		echo "regexp: $regexp"
		echo "Please see https://github.com/angular/angular.js/blob/master/DEVELOPERS.md#commit-message-format"
		exit 1
	fi

	echo "semantic commit title ok: $line"
done