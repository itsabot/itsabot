(function(abot) {
abot.Training = {}
abot.Training.controller = function() {
	var ctrl = this

	// Network request functions
	ctrl.getSentences = function() {
		ctrl.fetchAuthTokens()
		var id = ctrl.props.plugins()[ctrl.props.pluginIdx()].ID
		if (id === 0) {
			return
		}
		m.request({
			url: abot.itsAbotURL() + "/api/plugins/train/" + id,
			method: "GET",
		}).then(function(resp) {
			ctrl.props.published(true)
			var at = abot.state.authTokens()
			var trainable = false
			for (var i = 0; i < at.length; i++) {
				for (var j = 0; j < at[i].PluginIDs.length; j++) {
					if (at[i].PluginIDs[j] === id) {
						trainable = true
						break
					}
				}
			}
			ctrl.props.trainable(trainable)
			ctrl.props.sentences(resp)
			document.querySelector(".content").classList.remove("hidden")
		}, function(err) {
			if (err.Msg !== "plugin not published") {
				console.error(err)
			}
		})
	}
	ctrl.save = function() {
		var changed = document.querySelectorAll(".input-changed")
		if (changed.length === 0) {
			return
		}
		abot.externalRequest({
			url: abot.itsAbotURL() + "/api/plugins/train.json",
			method: "PUT",
			remotePluginID: ctrl.props.plugins()[ctrl.props.pluginIdx()].ID,
			data: ctrl.props.sentences().filter(function(s) {
				return s.Changed
			}),
		}).then(function(resp) {
			ctrl.props.error("")
			ctrl.props.success("Success! Saved changes.")
			for (var i = 0; i < changed.length; ++i) {
				changed[i].classList.remove("input-changed")
			}
		}, function(err) {
			ctrl.props.success("")
			ctrl.props.error(err.Msg)
		})
	}
	ctrl.deleteSentence = function(ev) {
		ev.preventDefault()
		if (confirm("Are you sure you want to delete this sentence?")) {
			var i = this.getAttribute("data-idx")
			var s = ctrl.props.sentences()[i]
			abot.externalRequest({
				url: abot.itsAbotURL() + "/api/plugins/train.json",
				method: "DELETE",
				remotePluginID: ctrl.props.plugins()[ctrl.props.pluginIdx()].ID,
				data: { Sentence: s.Sentence },
			}).then(function() {
				ctrl.props.sentences().splice(i, 1)
				ctrl.props.error("")
				ctrl.props.success("Success! Deleted training sentence.")
			}, function(err) {
				ctrl.props.success("")
				ctrl.props.error(err.Msg)
			})
		}
	}
	ctrl.fetchAuthTokens = function() {
		abot.request({
			url: "/api/admin/remote_tokens.json",
			method: "GET",
		}).then(abot.state.authTokens)
	}

	// UI functions and helpers
	ctrl.selectPlugin = function(id) {
		var plugin = ctrl.props.plugins()[id]
		ctrl.props.title("Training " + plugin.Name)
		ctrl.props.pluginIdx(id)
		ctrl.props.published(false)
		ctrl.props.sentences([])
		ctrl.getSentences()
	}
	ctrl.addSentence = function() {
		this.classList.add("hidden")
		document.getElementById("train-sentence").classList.remove("hidden")
	}

	ctrl.markChanged = function(val) {
		this.classList.add("input-changed")
		var i = this.getAttribute("data-idx")
		ctrl.props.sentences()[i].Intent = val
		ctrl.props.sentences()[i].Changed = true
	}
	ctrl.discard = function() {
		if (document.querySelector(".input-changed") == null) {
			return
		}
		if (confirm("Are you sure you want to discard your changes?")) {
			m.route(window.location.pathname, null, true)
		}
	}
	ctrl.props = {
		title: m.prop("Training"),
		plugins: m.prop([]),
		sentences: m.prop([]),
		published: m.prop(false),
		trainable: m.prop(false),
		error: m.prop(""),
		success: m.prop(""),

		// pluginIdx is the local index of the selected plugin, not the pluginID
		pluginIdx: m.prop(0),
	}
	ctrl.subviews = {
		newSentence: m.component(abot.TrainingNewSentence, ctrl),
	}

	// Set up the page, fetching initial data
	abot.Plugins.fetch().then(function(resp) {
		resp = resp || []

		// Plugin ordering is inconsistent. This sort fixes that.
		resp.sort(function(a, b) {
			return (a.Name > b.Name) ? 1 : ((b.Name > a.Name) ? -1 : 0);
		});
		ctrl.props.plugins(resp)
		ctrl.selectPlugin(0)
		ctrl.getSentences()
	}, function(err) {
		console.error(err)
	})
}
abot.Training.view = function(ctrl) {
	return m(".body", [
		m.component(abot.Header),
		m(".container", [
			m.component(abot.Sidebar, { active: 1 }),
			m(".main", [
				m(".topbar", [
					m(".topbar-inline", ctrl.props.title()),
					function() {
						var plugins = []
						var ps = ctrl.props.plugins()
						for (var i = 0; i < ps.length; ++i) {
							var el = m("a", {
								href: "#/",
								onclick: ctrl.selectPlugin.bind(this, i),
							}, ps[i].Name)
							plugins.push(el)
						}
						if (ps.length === 0) {
							return
						}
						return m(".topbar-right", plugins)
					}(),
				]),
				m(".content", [
					function() {
						if (ctrl.props.error().length > 0) {
							return m(".alert.alert-danger.alert-margin", ctrl.props.error())
						}
						if (ctrl.props.success().length > 0) {
							return m(".alert.alert-success.alert-margin", ctrl.props.success())
						}
					}(),
					function() {
						if (ctrl.props.plugins().length === 0) {
							return [
								m(".alert.alert-danger", "No plugins installed."),
								m("p", "Install a plugin to start training."),
							]
						} else if (!ctrl.props.published()) {
							var p = ctrl.props.plugins()[ctrl.props.pluginIdx()]
							return [
								m(".alert.alert-danger", "Publish this plugin to start training."),
								m("p", "To publish a plugin, enter the following command in your terminal (replacing your/go/get/path):"),
								m("code", "$ abot plugin publish your/go/get/path"),
							]
						} else {
							return [
								function() {
									if (ctrl.props.trainable()) {
										return m("h3.top-el", "Sentences")
									}
									return [
										m(".alert.alert-warn", [
											"This plugin's publisher hasn't connected their account to " + abot.itsAbotURL(),
											". Connect your account by following ",
											m("a[href=/account_connect]", { config: m.route }, "the instructions here."),
											" No changes you make here will be saved until the account is connected.",
										]),
										m("h3", "Sentences"),
									]
								}(),
								m("a[href=#]#train-sentence-btn", {
									onclick: ctrl.addSentence,
								}, "+ Train sentence"),
							]
						}
					}(),
					ctrl.subviews.newSentence,
					function() {
						var s = ctrl.props.sentences()
						var sentences = []
						for (var i = 0; i < s.length; ++i) {
							sentences.push(m(".list-item", [
								m("div", [
									m("a[href=#/].btn-x", {
										"data-idx": i,
										onclick: ctrl.deleteSentence
									}, "X"),
									s[i].Sentence,
								]),
								m(".badge-container.badge-container-right", [
									m(".badge", "Intent"),
									m("input[type=text]#intent.input-clear.input-sm", {
										"data-idx": i,
										placeholder: "set_alarm",
										value: s[i].Intent,
										oninput: m.withAttr("value", ctrl.markChanged),
									}),
								]),
							]))
						}
						return m("form.list", [
							sentences
						])
					}(),
					function() {
						if (ctrl.props.sentences().length === 0 || !ctrl.props.published()) {
							return
						}
						return m("#btns.btn-container-left", [
							m("input[type=button].btn", {
								onclick: ctrl.discard,
								value: "Discard Changes",
							}),
							m("input[type=button].btn.btn-primary", {
								onclick: ctrl.save,
								value: "Save",
							}),
						])
					}(),
				]),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
