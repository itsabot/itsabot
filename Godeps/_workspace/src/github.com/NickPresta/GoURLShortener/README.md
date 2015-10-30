# goisgd

A simple command line URL shortener using http://is.gd/.

## Getting the Code

    go get github.com/NickPresta/GoURLShortener

## Usage

Import this library in your code to use it. The package name is `goisgd`.

    import (
        "github.com/NickPresta/GoURLShortener"
    )

Shorten a URI:

    uri, err := goisgd.Shorten("http://google.ca/")

See `examples/main.go` for an example.

## Documentation

View the documentation on
[GoPkgDoc](http://go.pkgdoc.org/github.com/NickPresta/GoURLShortener).

## Tests

There are no tests.

## Changelog

See `CHANGELOG.md` for details.

## License

This is released under the
[MIT license](http://www.opensource.org/licenses/mit-license.php).
See `LICENSE.md` for details.

![Powered by Gophers](http://i.imgur.com/SwkPj.png "Powered by Gophers")
