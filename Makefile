# Copyright 2014 tsuru-admin authors. All rights reserved.
# Use of this source code is governed by a BSD-style
#  license that can be found in the LICENSE file.

release:
	@echo "Releasing tsuru-admin $(version) version."

	@echo "Replacing version string."
	@sed -i "" "s/version = \".*\"/version = \"$(version)\"/g" main.go

	@git add .
	@git commit -m "bump to $(version)"

	@echo "Creating $(version) tag."
	@git tag $(version)

	@git push --tags
	@git push origin master

	@echo "$(version) released!"
