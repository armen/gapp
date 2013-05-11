BUILD = $(shell date +'%Y%m%d%H%M%S')

install:
	go get gapp

comp-assets:
	echo ${BUILD} > conf/BUILD
	git clean -fX static/css
	lessc -x --yui-compress assets/less/bootstrap.less static/css/bootstrap.${BUILD}.css
	lessc -x --yui-compress assets/less/responsive.less static/css/bootstrap-responsive.${BUILD}.css
	lessc -x --yui-compress assets/less/gapp.less static/css/gapp.${BUILD}.css
	git clean -fX static/js/min
	uglifyjs assets/js/jquery-1.7.2.js > static/js/jquery-1.7.2.${BUILD}.js
	uglifyjs assets/js/gapp.js > static/js/gapp.${BUILD}.js
	uglifyjs assets/js/bootstrap-tooltip.js > static/js/bootstrap-tooltip.${BUILD}.js
	uglifyjs assets/js/bootstrap-popover.js > static/js/bootstrap-popover.${BUILD}.js

dev-assets:
	echo ${BUILD} > conf/BUILD
	git clean -fX static/css
	lessc assets/less/bootstrap.less static/css/bootstrap.${BUILD}.css
	lessc assets/less/responsive.less static/css/bootstrap-responsive.${BUILD}.css
	lessc assets/less/gapp.less static/css/gapp.${BUILD}.css
	git clean -fX static/js
	ln -sf ../../assets/js/jquery-1.7.2.js static/js/jquery-1.7.2.${BUILD}.js
	ln -sf ../../assets/js/gapp.js static/js/gapp.${BUILD}.js
	ln -sf ../../assets/js/bootstrap-tooltip.js static/js/bootstrap-tooltip.${BUILD}.js
	ln -sf ../../assets/js/bootstrap-popover.js static/js/bootstrap-popover.${BUILD}.js

gofmt-hook:
	wget -q https://raw.github.com/edsrzf/gofmt-git-hook/master/fmt-check -O /tmp/pre-commit
	mv --interactive /tmp/pre-commit $(shell git rev-parse --git-dir)/hooks/
	chmod +x $(shell git rev-parse --git-dir)/hooks/pre-commit
