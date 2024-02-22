.PHONY: count
count:
	@ find . -name tests -prune -o -type f -name '*.go' | xargs wc -l