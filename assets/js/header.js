(function(abot) {
abot.Header = {}
abot.Header.controller = function() {
	var ctrl = this
	ctrl.signout = function(ev) {
		ev.preventDefault()
		cookie.removeItem("id")
		cookie.removeItem("trainer")
		cookie.removeItem("session_token")
		m.route("/login")
	}
}
abot.Header.view = function(ctrl) {
	var tour = null
	var profileOrLogin
	if (cookie.getItem("id") !== null) {
		profileOrLogin = [
			m("a[href=/profile]", {
				config: m.route
			}, "Profile"),
			m("a[href=#/]", {
				onclick: ctrl.signout,
			}, "Sign out"),
		]
	} else {
		profileOrLogin = [
			m("a[href=/login]", {
				config: m.route
			}, "Log in"),
			m("a[href=/signup]", {
				config: m.route
			}, "Sign up"),
		]
	}
	if (cookie.getItem("id") === null || m.route() === "/") {
		tour = m("a", {
			href: "/tour",
			config: m.route
		}, "Tour")
	}
	return m("header", [
		m("div", [
			m(".links", [
				m("a", {
					href: "/",
					config: m.route
				}, "Home"),
				profileOrLogin
			]),
			m(".logo", [
				m("a", {
					href: "/",
					config: m.route
				}, "Abot")
			])
		]),
		m("div", { id: "content" })
	])
}
})(!window.abot ? window.abot={} : window.abot);
