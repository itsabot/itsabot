(function(ava) {
ava.isFunction = function(obj) {
	return Object.prototype.toString.call(obj) === "[object Function]"
}
ava.fnCopy = function(from, to) {
	for (var i = 2; i < arguments.length; i++) {
		fr = from[arguments[i]]
		if (!!fr && ava.isFunction(fr) && !to[arguments[i]]) {
			to[arguments[i]] = fr
		}
	}
}
ava.isLoggedIn = function() {
	var userId = cookie.getItem("id")
	return userId != null && parseInt(userId) > 0
}
ava.prettyDate = function(time) {
    var date = new Date((time || "").replace(/-/g, "/").replace(/[TZ]/g, " ")),
        diff = (((new Date()).getTime() - date.getTime()) / 1000),
        day_diff = Math.floor(diff / 86400)

    if (isNaN(day_diff) || day_diff < 0 || day_diff >= 31) return

    return day_diff == 0 && (
	diff < 60 && "just now" || diff < 120 && "1 minute ago" || diff < 3600 && Math.floor(diff / 60) + " minutes ago" || diff < 7200 && "1 hour ago" || diff < 86400 && Math.floor(diff / 3600) + " hours ago") || day_diff == 1 && "Yesterday" || day_diff < 7 && day_diff + " days ago" || day_diff < 31 && Math.ceil(day_diff / 7) + " weeks ago"
}
ava.auth2Callback = function() {
	ava.auth2 = gapi.auth2.getAuthInstance()
	if (!!ava.auth2) {
		return
	}
	var gid = document.querySelector("meta[name=google-client-id]").getAttribute("content")
	gapi.auth2.init({
		client_id: gid,
		scope: "https://www.googleapis.com/auth/calendar"
	}).then(function(a) {
		ava.auth2 = a
		if (ava.auth2.isSignedIn.get()) {
			var email = ava.auth2.currentUser.get().getBasicProfile().getEmail()
			ava.toggleGoogleAccount(email)
		}
	}, function(err) {
		console.error(err)
	})
}
ava.toggleGoogleAccount = function(name) {
	var googleLink = document.getElementById("oauth-google-success-a")
	if (!googleLink) {
		// Not on the Profile page. This function is called globally on
		// Google's script loading, so it isn't dependent on any route.
		// Ultimately Google's script should only load on the Profile route,
		// which eliminates the need for this check
		return
	}
	var signout = document.getElementById("oauth-google-success")
	var signin = document.getElementById("signinButton")
	if (!name) {
		googleLink.text = ""
		signout.classList.add("hidden")
		signin.classList.remove("hidden")
	} else {
		googleLink.text = "Google - " + name
		signout.classList.remove("hidden")
		signin.classList.add("hidden")
	}
}
window.addEventListener('load', function() {
	gapi.load("auth2", ava.auth2Callback)
	m.route.mode = "pathname"
	m.route(document.body, "/", {
		"/": ava.Index,
		"/tour": ava.Tour,
		"/train": ava.TrainIndex,
		"/train/:id": ava.TrainShow,
		"/signup": ava.Signup,
		"/login": ava.Login,
		"/forgot_password": ava.ForgotPassword,
		"/reset_password": ResetPassword,
		"/profile": ava.Profile,
		"/cards/new": ava.NewCard,
	})
})
})(!window.ava ? window.ava={} : window.ava);
