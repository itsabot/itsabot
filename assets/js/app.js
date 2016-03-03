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
abot.signout = function(ev) {
	ev.preventDefault()
	abot.request({
		url: "/api/logout.json",
		method: "POST",
	}).then(function() {
		cookie.removeItem("id")
		cookie.removeItem("email")
		cookie.removeItem("issuedAt")
		cookie.removeItem("scopes")
		cookie.removeItem("csrfToken")
		cookie.removeItem("authToken")
		m.route("/login")
	}, function(err) {
		console.error(err)
	})
}
abot.isLoggedIn = function() {
	if (cookie.getItem("id") != null &&
		cookie.getItem("email") != null &&
		cookie.getItem("issuedAt") != null &&
	    cookie.getItem("authToken") != null) {
		return true
	}
	// If the user isn't logged in, ensure we clean out all cookies.
	cookie.setItem("id", null)
	cookie.setItem("email", null)
	cookie.setItem("issuedAt", null)
	cookie.setItem("scopes", null)
	cookie.setItem("authToken", null)
	return false
}
abot.isTrainer = function() {
	var scopes = cookie.getItem("scopes")
	if (scopes == null) {
		return false
	}
	scopes = scopes.split(" ")
	for	(var i = 0; i < scopes.length; ++i) {
		if (scopes[i] === "trainer") {
			return true
		}
	}
	return false
}
abot.request = function(opts) {
	opts.config = function(xhr) {
		xhr.setRequestHeader("Authorization", "Bearer " + cookie.getItem("authToken"))
		xhr.setRequestHeader("X-CSRF-Token", cookie.getItem("csrfToken"))
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
	})
})
})(!window.abot ? window.abot={} : window.abot);
