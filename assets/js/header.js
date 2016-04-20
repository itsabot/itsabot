(function(abot) {
abot.Header = {}
abot.Header.view = function() {
	var tour = null
	var profileOrLogin
	if (Cookies.get("id") !== null) {
		profileOrLogin = [
			m("a[href=/profile]", {
				config: m.route
			}, "Profile"),
			m("a[href=#/]", {
				onclick: abot.signout,
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
	var admin
	if (abot.isAdmin()) {
		admin = m("a[href=/admin]", { config: m.route }, "Admin")
	}
	return m("header", [
		m("div", [
			m(".links", [
				m("a", {
					href: "/",
					config: m.route
				}, "Home"),
				admin,
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
