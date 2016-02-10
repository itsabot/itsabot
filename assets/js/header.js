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
	return m("header", { class: m.route()==="/"?"":"gradient" }, [
		m("div", { class: "container" }, [
			m("a", {
				class: "navbar-brand",
				href: "/",
				config: m.route
			}, [
				m("div", [
					m("img", {
						src: "/public/images/logo.svg"
					}),
					m("span", {
						class: "margin-top-xs hide-small"
					}, m.trust(" &nbsp;Ava")),
				])
			]),
			m("div", { class: "text-right navbar-right" }, [
				m("a", {
					href: "/",
					config: m.route
				}, "Home"),
				tour,
				m("a", {
					href: "https://medium.com/ava-updates/latest",
					class: "hide-small"
				}, "Updates"),
				profileOrLogin
			])
		]),
		m("div", { id: "content" })
	])
}
})(!window.ava ? window.ava={} : window.ava);
