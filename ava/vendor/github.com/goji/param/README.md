Param [![GoDoc](https://godoc.org/github.com/goji/param?status.svg)](https://godoc.org/github.com/goji/param)
=====

param deserializes parameter values into a given struct using magical
reflection ponies.

Inspired by gorilla/schema, but uses Rails/jQuery style param
encoding instead of their weird dotted syntax. In particular, this package was
written with the intent of parsing the output of jQuery.param.

This package uses struct tags to guess what names things ought to have. If a
struct value has a "param" tag defined, it will use that. If there is no "param"
tag defined, the name part of the "json" tag will be used. If that is not
defined, the name of the field itself will be used (no case transformation is
performed).

If the name derived in this way is the string "-", param will refuse to set that
value.

The parser is extremely strict, and will return an error if it has any
difficulty whatsoever in parsing any parameter, or if there is any kind of type
mismatch.

## Example

Here's how to use `param` to parse the contents of a web form:

```go

import (
    "net/http"
    "github.com/goji/param"
)

type SignupForm struct {
    Name string `param:"name"`
    Email string `param:"email_address"`
    // We use a struct tag with "-" to ignore a value.
    Password string `param:"-"`
}

// FormHandler accepts a POST request, and would typically handle a HTML 
// form with a format like this:
// 
//  <form action="/signup/submit" method="POST">
//  <input name="name" type="text">
//  <input name="email_address" type="text">
//  <input name="password" type="password">
//  <input type="submit" value="Signup!">
//  </form>
//
// The 'name' attributes should match up with those of our struct fields. If 
// they don't, we use the aforementioned struct tags to translate them.
func FormHandler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if err != nil {
        http.Error(w, "No good!", 400)
        return
    }

    var signupForm SignupForm{}
    // Parse url.Values (in this case, r.PostForm) and 
    // a pointer to our struct so that param can populate it.
    err := param.Parse(r.PostForm, &signupForm)
    if err != nil {
        http.Error(w, "Real bad.", 500)
        return
    }

    // Now we can:
    // - Perform some validation on our values
    // - Hash the user password with bcrypt or scrypt
    // - Store the results in our database
    // - (the world is our oyster!)
}

```

It's pretty simple! Note that you can also inspect the errors returned from `param.Parse` if you wish. Error types are documented [over on GoDoc](http://godoc.org/github.com/goji/param#pkg-index).

## License

MIT licensed. See the LICENSE file for details.
