#!/bin/sh

fswatch -0 -o post-social.go | xargs -0 -n1 ./afterwatch.sh
