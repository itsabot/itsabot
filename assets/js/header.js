(function(abot) {
abot.Header = {}
abot.Header.view = function() {
	var tour = null
	var profileOrLogin
	if (abot.isLoggedIn()) {
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
	return m("header", [
		m("div", [
			m(".links", [
				profileOrLogin
			]),
			m(".logo", [
				m("a.logo", {
					href: "/",
					config: m.route
				}, m("img[src=/public/images/logo_white.svg]", {
					alt: "Abot",
				})),
			]),
		]),
		m("div", { id: "content" })
	])
}
})(!window.abot ? window.abot={} : window.abot);
