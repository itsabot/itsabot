(function(ava) {
ava.Header = {}
ava.Header.view = function() {
	var tour = null
	var profileOrLogin
	if (cookie.getItem("id") !== null) {
		profileOrLogin = m("a", {
			href: "/profile",
			config: m.route
		}, "Profile")
	} else {
		profileOrLogin = m("a", {
			href: "/login",
			config: m.route
		}, "Log in")
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
})(!window.ava ? window.ava={} : window.ava);
