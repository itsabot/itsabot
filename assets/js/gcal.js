(function(ava) {
ava.Gcal = {}
ava.Gcal.controller = function() {
	var ctrl = this
	ctrl.googleOAuth = function(ev) {
		ev.preventDefault()
		ava.auth2.grantOfflineAccess({'redirect_uri': 'postmessage'})
		.then(ctrl.signInCallback, function(err) {
			console.log("ERR HERE")
			console.log(err)
		})
	}
	ctrl.signInCallback = function(authResult) {
		if (authResult["code"]) {
			m.request({
				method: "POST",
				url: window.location.origin + "/oauth/connect/gcal.json",
				data: {
					Code: authResult["code"],
					UserID: parseInt(cookie.getItem("id")),
				}
			}).then(function() {
				var email = ava.auth2.currentUser.get().getBasicProfile().getEmail()
				ava.toggleGoogleAccount(email)
			}, function(err) {
				console.error(err)
			})
		} else {
			console.error("something went wrong")
		}
	}
	ctrl.googleLink = function() {
		return document.getElementById("oauth-google-success-a")
	}
	ctrl.googleRevoke = function(ev) {
		if (!!ev) { ev.preventDefault() }
		if (confirm("Disconnect Google?")) {
			ava.auth2.disconnect()
			ava.auth2.signOut()
			ava.toggleGoogleAccount()
		}
	}
}
ava.Gcal.view = function(ctrl) {
	return m('div', [
		m("h3.margin-top-sm", "Calendars"),
		m(".form-group.card", [
			m("div", [
				m("#oauth-google-success", [
					m("a#oauth-google-success[href=#/]", {
						class: "hidden",
						onclick: ctrl.googleRevoke
					}, "Google")
				]),
				m("input", {
					id: "signinButton",
					class: "btn-oauth-signin",
					type: "image",
					src: "/public/images/btn_google_signin_light_normal_web.png",
					onclick: ctrl.googleOAuth
				}, "Sign in with Google"),
			])
		])
	])
}
})(!window.ava ? window.ava={} : window.ava);
