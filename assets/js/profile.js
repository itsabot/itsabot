(function(ava) {
ava.Profile = {}
ava.Profile.controller = function() {
	var userId = cookie.getItem("id")
	if (!userId || userId <= 0) {
		cookie.removeItem("id")
		cookie.removeItem("trainer")
		cookie.removeItem("session_token")
		return m.route("/login")
	}
	var redirect = m.route.param("r")
	if (!!redirect) {
		m.route("/" + redirect.substring(1))
	}
	var ctrl = this
	ava.loadJS("https://apis.google.com/js/client.js?onload", function() {
		gapi.load("auth2", ctrl.auth2Callback)
	})
	ctrl.signout = function(ev) {
		ev.preventDefault()
		cookie.removeItem("id")
		cookie.removeItem("session_token")
		m.route("/login")
	},
	ctrl.data = function(uid) {
		return m.request({
			method: "GET",
			url: "/api/profile.json?uid=" + uid
		})
	},
	ctrl.sendView = function(uid) {
		return m.request({
			method: "PUT",
			url: "/api/profile.json",
			data: { UserID: parseInt(uid, 10) }
		})
	}
	ctrl.auth2Callback = function() {
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
				ctrl.toggleGoogleAccount(email)
			}
		}, function(err) {
			console.error(err)
		})
	}
	ctrl.toggleGoogleAccount = function(name) {
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
	ctrl.props = {
		username: m.prop(""),
		email: m.prop(""),
		phones: m.prop([]),
		cards: m.prop([])
	}
	ctrl.data(userId).then(function(data) {
		ctrl.props.email(data.Email)
		ctrl.props.username(data.Name)
		ctrl.props.phones(data.Phones || [])
		ctrl.props.cards(data.Cards || [])
		var c = {
			id: 123,
			CardholderName: 'Jared Borner',
			Brand: 'Visa',
			ExpMonth: '7',
			ExpYear: '17',
			Last4: 1234,
		}
		ctrl.props.cards().push(c)
	}, function(err) {
		console.error(err)
	})
	ctrl.sendView(userId)
}
ava.Profile.view = function(ctrl) {
	return m(".body", [
		m.component(ava.Header),
		ava.Profile.viewFull(ctrl),
		m.component(ava.Footer)
	])
}
ava.Profile.viewFull = function(ctrl) {
	return m("#full.container", [
		m(".row.margin-top-sm", m(".col-md-12", m("h1", "Profile"))),
		m(".row", [
			m(".col-md-7.margin-top-sm", [
				m("h3", "Account Details"),
				m("form.margin-top-sm", [
					m(".card", [
						m(".form-group", [
							m("label", "Username"),
							m("div", m("div", ctrl.props.email()))
						]),
						m(".form-group", [
							m("label", "Password"),
							m("div", m("a[href=#]", "Change password"))
						]),
						m(".form-group", [
							m("label", {
								for: "username"
							}, "Name"),
							m("div", [
								m("input", {
									id: "username",
									type: "text",
									class: "form-control",
									value: ctrl.props.username()
								})
							])
						]),
						m(".form-group.margin-top-sm", [
							m("div", [
								m("a", {
									class: "btn btn-sm",
									href: "#/",
									onclick: ctrl.signout,
								}, "Sign out")
							])
						])
					]),
					m.component(ava.Phones, ctrl.props.phones()),
					m.component(ava.Cards, ctrl.props.cards()),
					m.component(ava.Gcal)
				])
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
