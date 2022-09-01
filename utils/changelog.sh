#!/bin/bash
# Generate a Markdown change log of pull requests from commits between two tags
# Author: Russell Heimlich
# URL: https://gist.github.com/kingkool68/09a201a35c83e43af08fcbacee5c315a

# HOW TO USE
# Copy this script to a directory under Git version control
# Make the script executable i.e. chmod +x changelog.sh
# Run it! ./changelog.sh
# Check CHANGELOG.md to see your results

# Repo URL to base links off of
REPOSITORY_URL=https://github.com/c9s/bbgo

# Get a list of all tags in reverse order
# Assumes the tags are in version format like v1.2.3
GIT_TAGS=$(git tag -l --sort=-version:refname)

# Make the tags an array
TAGS=($GIT_TAGS)
LATEST_TAG=${TAGS[0]}
PREVIOUS_TAG=${TAGS[1]}

# If you want to specify your own two tags to compare, uncomment and enter them below
PREVIOUS_TAG=$LATEST_TAG
LATEST_TAG=main

# Get a log of commits that occured between two tags
# We only get the commit hash so we don't have to deal with a bunch of ugly parsing
# See Pretty format placeholders at https://git-scm.com/docs/pretty-formats
COMMITS=$(git log $PREVIOUS_TAG..$LATEST_TAG --pretty=format:"%H")

# Store our changelog in a variable to be saved to a file at the end
MARKDOWN="[Full Changelog]($REPOSITORY_URL/compare/$PREVIOUS_TAG...$LATEST_TAG)"
MARKDOWN+='\n'

# Loop over each commit and look for merged pull requests
for COMMIT in $COMMITS; do
	# Get the subject of the current commit
	SUBJECT=$(git log -1 ${COMMIT} --pretty=format:"%s")

	# If the subject contains "Merge pull request #xxxxx" then it is deemed a pull request
	PULL_REQUEST=$( grep -Eo "Merge pull request #[[:digit:]]+" <<< "$SUBJECT" )
	if [[ $PULL_REQUEST ]]; then
		# Perform a substring operation so we're left with just the digits of the pull request
		PULL_NUM=${PULL_REQUEST#"Merge pull request #"}
		# AUTHOR_NAME=$(git log -1 ${COMMIT} --pretty=format:"%an")
		# AUTHOR_EMAIL=$(git log -1 ${COMMIT} --pretty=format:"%ae")

		# Get the body of the commit
		BODY=$(git log -1 ${COMMIT} --pretty=format:"%b")
		MARKDOWN+='\n'
		MARKDOWN+=" - [#$PULL_NUM]($REPOSITORY_URL/pull/$PULL_NUM): $BODY"
	fi
done

# Save our markdown to a file
echo -e $MARKDOWN
