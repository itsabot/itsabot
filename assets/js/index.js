(function(abot) {
abot.Index = {}
abot.Index.controller = function() {
	if (abot.isAdmin()) {
		return m.route("/admin", null, true)
	}
	if (abot.isLoggedIn()) {
		return m.route("/profile", null, true)
	}
	m.request({
		url: "/api/admin_exists.json",
		method: "GET",
	}).then(function(resp) {
		if (resp === true) {
			return m.route("/login", null, true)
		}
		abot.Index.view = abot.Index.loadedView
	}, function(err) {
		console.log(err.Msg)
	})
}
abot.Index.view = function() {
	return m(".container", [
		m.component(abot.Header),
	])
}
abot.Index.loadedView = function() {
	return m(".container", [
		m.component(abot.Header),
		m(".main-no-sidebar", [
			m(".centered", [
				m("img[src=/public/images/logo.svg].big-icon"),
			]),
			m(".well.well-form", [
				m(".centered", [ m("h3.top-el", "Welcome to Abot!") ]),
				m("p.top-el", [
					"Before you start building your bot, let's set up an admin account.",
				]),
				m("form.centered", [
					m(".form-el", m("input[type=text][placeholder=Name]", { config: this.focus })),
					m(".form-el", m("input[type=email][placeholder=Email]")),
					m(".form-el", m("input[type=tel][placeholder=Phone Number]")),
					m(".form-el", m("input[type=password][placeholder=Password]")),
					m(".form-el", m("input[type=submit][value=\"Create Admin Account\"].btn")),
				]),
			]),
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
