(function(abot) {
abot.isFunction = function(obj) {
	return Object.prototype.toString.call(obj) === "[object Function]"
}
abot.fnCopy = function(from, to) {
	for (var i = 2; i < arguments.length; i++) {
		fr = from[arguments[i]]
		if (!!fr && abot.isFunction(fr) && !to[arguments[i]]) {
			to[arguments[i]] = fr
		}
	}
}
abot.isProduction = function() {
	var ms = document.getElementsByTagName("meta")
	for (var i = 0; i < ms.length; i++) {
		if (ms[i].getAttribute("name") === "env-production") {
			return ms[i].getAttribute("content") === "true"
		}
	}
	return false
}
abot.signout = function(ev) {
	ev.preventDefault()
	abot.request({
		url: "/api/logout.json",
		method: "POST",
	}).then(function() {
		Cookies.expire("id")
		Cookies.expire("email")
		Cookies.expire("issuedAt")
		Cookies.expire("scopes")
		Cookies.expire("csrfToken")
		Cookies.expire("authToken")
		m.route("/login")
	}, function(err) {
		console.error(err)
	})
}
abot.isLoggedIn = function() {
	if (Cookies.get("id") != null &&
		Cookies.get("email") != null &&
		Cookies.get("issuedAt") != null &&
	    Cookies.get("authToken") != null) {
		return true
	}
	// If the user isn't logged in, ensure we clean out all cookies.
	Cookies.expire("id", null)
	Cookies.expire("email", null)
	Cookies.expire("issuedAt", null)
	Cookies.expire("scopes", null)
	Cookies.expire("authToken", null)
	return false
}
abot.isAdmin = function() {
	var scopes = Cookies.get("scopes")
	if (scopes == null) {
		return false
	}
	scopes = scopes.split(" ")
	for	(var i = 0; i < scopes.length; ++i) {
		if (scopes[i] === "admin") {
			return true
		}
	}
	return false
}
abot.request = function(opts) {
	opts.config = function(xhr) {
		xhr.setRequestHeader("Authorization", "Bearer " + Cookies.get("authToken"))
		xhr.setRequestHeader("X-CSRF-Token", Cookies.get("csrfToken"))
	}
	return m.request(opts)
}
abot.prettyDate = function(time) {
    var date = new Date((time || "").replace(/-/g, "/").replace(/[TZ]/g, " ")),
        diff = (((new Date()).getTime() - date.getTime()) / 1000),
        day_diff = Math.floor(diff / 86400)
    if (isNaN(day_diff) || day_diff < 0 || day_diff >= 31) return
    return day_diff == 0 && (
		diff < 60 && "just now" ||
		diff < 120 && "1 minute ago" ||
		diff < 3600 && Math.floor(diff / 60) + " minutes ago" ||
		diff < 7200 && "1 hour ago" ||
		diff < 86400 && Math.floor(diff / 3600) + " hours ago") ||
		day_diff == 1 && "Yesterday" ||
		day_diff < 7 && day_diff + " days ago" ||
		day_diff < 31 && Math.ceil(day_diff / 7) + " weeks ago"
}
abot.loadJS = function(url, cb) {
	if (document.getElementById(url) !== null) {
		return
	}
	var s = document.createElement("script")
	s.src = url
	s.id = url
	s.onload = cb
	document.head.appendChild(s)
}
window.addEventListener('load', function() {
	m.route.mode = "pathname"
	m.route(document.body, "/", {
		"/": abot.Index,
		"/signup": abot.Signup,
		"/login": abot.Login,
		"/forgot_password": abot.ForgotPassword,
		"/reset_password": abot.ResetPassword,
		"/profile": abot.Profile,
		"/admin": abot.Admin,
	})
})
})(!window.abot ? window.abot={} : window.abot);
