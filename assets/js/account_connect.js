(function(abot) {
abot.AccountConnect = {}
abot.AccountConnect.controller = function() {
	abot.AccountConnect.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile")
		}
	})
	var ctrl = this
	ctrl.submitAuthToken = function(ev) {
		ev.preventDefault()
		ctrl.props.success("")
		ctrl.props.error("")
		ctrl.testAuthToken(function(valid, plugins) {
			if (!valid) {
				return
			}
			abot.request({
				url: "/api/admin/remote_tokens.json",
				method: "POST",
				data: {
					Token: ctrl.props.token(),
					PluginIDs: plugins,
				},
			}).then(function(resp) {
				var token = {
					Token: ctrl.props.token(),
					Email: Cookies.get("email"),
				}
				ctrl.props.tokens().push(token)
				ctrl.props.token("")
				ctrl.props.success("Success! Added auth token.")
			}, function(err) {
				ctrl.props.error(err.Msg)
			})
		})
	}
	ctrl.testAuthToken = function(cb) {
		m.request({
			url: abot.itsAbotURL() + "/api/plugins/test_auth.json",
			method: "POST",
			data: { Token: ctrl.props.token() },
		}).then(function(resp) {
			cb(true, resp)
		}, function(err) {
			ctrl.props.success("")
			ctrl.props.error(err.Msg)
			cb(false)
		})
	}
	ctrl.fetchAuthTokens = function() {
		abot.request({
			url: "/api/admin/remote_tokens.json",
			method: "GET",
		}).then(function(resp) {
			ctrl.props.tokens(resp)
		}, function(err) {
			ctrl.props.error(err.Msg)
		})
	}

	// Prepare component state
	ctrl.props = {
		token: m.prop(""),
		tokens: m.prop([]),

		// Flash messages related to the account connections
		success: m.prop(""),
		error: m.prop(""),
	}
	var a = document.createElement("a")
	a.href = abot.itsAbotURL()
	ctrl.hostname = a.hostname

	// Fetch data
	ctrl.fetchAuthTokens()
}
abot.AccountConnect.view = function(ctrl) {
	return m(".container", [
		m.component(abot.Header),
		m.component(abot.Sidebar, { active: 4 }),
		m(".main", [
			m(".topbar", "Account Connect"),
			m(".content", [
				function() {
					if (ctrl.props.error().length > 0) {
						return m(".alert.alert-danger.alert-margin", ctrl.props.error())
					}
					if (ctrl.props.success().length > 0) {
						return m(".alert.alert-success.alert-margin", ctrl.props.success())
					}
				}(),
				m("form", { onsubmit: ctrl.submitAuthToken }, [
					m("div", [
						"Connecting your account to ",
						m("strong", ctrl.hostname),
						" will enable you to train plugins you've published. If you've published a plugin to " + ctrl.hostname + ", then generate an auth token at ",
						m("a", {
							href: abot.itsAbotURL() + "/profile",
						}, abot.itsAbotURL() + "/profile"),
						" and paste it here to authenticate yourself."
					]),
					m("p", "This has to be done once for each plugin publisher."),
					m(".form-el", [
						m("input[type=text]", {
							placeholder: "Auth Token",
							value: ctrl.props.token(),
							onchange: m.withAttr("value", ctrl.props.token),
						}),
						m("input.btn.btn-inline[type=submit]", {
							value: "Connect Account",
						}),
					]),
				]),
				function() {
					if (ctrl.props.tokens().length === 0) {
						return
					}
					return m("div", [
						m("h4", "Account Connect Tokens"),
						m("table.table-compact", [
							m("thead", [
								m("th", ""),
								m("th", "Auth Token (Last 6)"),
								m("th", "Added By"),
							]),
							function() {
								var t = ctrl.props.tokens()
								var els = []
								for (var i = 0; i < t.length; i++) {
									els.push(m.component(abot.TableItemToken, ctrl, t[i]))
								}
								return els
							}(),
						]),
					])
				}(),
			]),
		]),
	])
}
abot.AccountConnect.checkAuth = function(callback) {
	var at = Cookies.get("remoteAuthToken")
	return callback(at != null && at !== "undefined")
}
})(!window.abot ? window.abot={} : window.abot);
