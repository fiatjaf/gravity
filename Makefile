all: static/bundle.js static/style.css static/ReactToastify.min.css

prod: static/bundle.min.js static/style.min.css static/ReactToastify.min.css

watch:
	find client -name '*.js' -o -name '*.styl' | entr make

static/bundle.js: $(shell find client/)
	./node_modules/.bin/browserify client/app.js -dv --outfile static/bundle.js

static/bundle.min.js:  $(shell find client/)
	./node_modules/.bin/browserify client/app.js -g [ envify --NODE_ENV production ] -g uglifyify | ./node_modules/.bin/uglifyjs --compress --mangle > static/bundle.min.js

static/style.css: client/style.styl
	./node_modules/.bin/stylus < client/style.styl > static/style.css

static/style.min.css: client/style.styl
	./node_modules/.bin/stylus -c < client/style.styl > static/style.css

static/ReactToastify.min.css: node_modules/react-toastify
	cp node_modules/react-toastify/dist/ReactToastify.min.css static/
