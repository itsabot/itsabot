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
ava.isTrainer = function() {
	var trainer = cookie.getItem("trainer")
	return trainer != null && Boolean(trainer)
}
ava.prettyDate = function(time) {
    var date = new Date((time || "").replace(/-/g, "/").replace(/[TZ]/g, " ")),
        diff = (((new Date()).getTime() - date.getTime()) / 1000),
        day_diff = Math.floor(diff / 86400)

    if (isNaN(day_diff) || day_diff < 0 || day_diff >= 31) return

    return day_diff == 0 && (
	diff < 60 && "just now" || diff < 120 && "1 minute ago" || diff < 3600 && Math.floor(diff / 60) + " minutes ago" || diff < 7200 && "1 hour ago" || diff < 86400 && Math.floor(diff / 3600) + " hours ago") || day_diff == 1 && "Yesterday" || day_diff < 7 && day_diff + " days ago" || day_diff < 31 && Math.ceil(day_diff / 7) + " weeks ago"
}
ava.loadJS = function(url, cb) {
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
