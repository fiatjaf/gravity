all: static/bundle.js static/style.css static/ReactToastify.min.css

prod: static/bundle.min.js static/style.min.css static/ReactToastify.min.css

cmd: cmd/gravity/gravity_linux_386 cmd/gravity/gravity_linux_amd64 cmd/gravity/gravity_darwin_amd64 cmd/gravity/gravity_windows_386 cmd/gravity/gravity_windows_amd64

cmd/gravity/gravity_linux_386: $(shell find cmd/gravity/ -name '*.go')
	cd cmd/gravity/ && gox -osarch="linux/386"
cmd/gravity/gravity_linux_amd64: $(shell find cmd/gravity/ -name '*.go')
	cd cmd/gravity/ && gox -osarch="linux/amd64" 
cmd/gravity/gravity_darwin_amd64: $(shell find cmd/gravity/ -name '*.go')
	cd cmd/gravity/ && gox -osarch="darwin/amd64" 
cmd/gravity/gravity_windows_386: $(shell find cmd/gravity/ -name '*.go')
	cd cmd/gravity/ && gox -osarch="windows/386" 
cmd/gravity/gravity_windows_amd64: $(shell find cmd/gravity/ -name '*.go')
	cd cmd/gravity/ && gox -osarch="windows/amd64"

watch:
	find client -name '*.js' -o -name '*.styl' | entr make

static/bundle.js: $(shell find client/)
	godotenv -f .env ./node_modules/.bin/browserify client/app.js -dv --outfile static/bundle.js

static/bundle.min.js:  $(shell find client/)
	./node_modules/.bin/browserify client/app.js -g [ envify --NODE_ENV production ] -g uglifyify | ./node_modules/.bin/uglifyjs --compress --mangle > static/bundle.min.js

static/style.css: client/style.styl
	./node_modules/.bin/stylus < client/style.styl > static/style.css

static/style.min.css: client/style.styl
	./node_modules/.bin/stylus -c < client/style.styl > static/style.css

static/ReactToastify.min.css: node_modules/react-toastify
	cp node_modules/react-toastify/dist/ReactToastify.min.css static/
