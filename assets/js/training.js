(function(abot) {
abot.Training = {}
abot.Training.controller = function() {
	var ctrl = this
	ctrl.selectPlugin = function(id) {
		// Change title, change sentences
		var plugin = ctrl.props.plugins()[id]
		ctrl.props.title("Train " + plugin.Name)
		ctrl.props.pluginIdx(id)
		ctrl.props.published(false)
		ctrl.props.sentences([])
		ctrl.getSentences()
	}
	ctrl.addSentence = function() {
		this.classList.add("hidden")
		document.getElementById("train-sentence").classList.remove("hidden")
	}
	ctrl.getSentences = function() {
		var plg = ctrl.props.plugins()[ctrl.props.pluginIdx()].Name
		plg = encodeURIComponent(plg)
		m.request({
			url: abot.itsAbotURL() + "/api/plugins/train/" + plg,
			method: "GET",
		}).then(function(resp) {
			ctrl.props.published(true)
			ctrl.props.sentences(resp || [])
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
		var s = ctrl.props.sentences().filter(function(s) {
			return s.Changed === true
		})
		m.request({
			url: abot.itsAbotURL() + "/api/plugins/train.json",
			method: "PUT",
			data: ctrl.props.sentences().filter(function(s) {
					return s.Changed === true
			}),
		})
		for (var i = 0; i < changed.length; ++i) {
			changed[i].classList.remove("input-changed")
		}
	}
	ctrl.deleteSentence = function(ev) {
		ev.preventDefault()
		if (confirm("Are you sure you want to delete this sentence?")) {
			var i = this.getAttribute("data-idx")
			var s = ctrl.props.sentences().splice(i, 1)[0]
			m.request({
				url: abot.itsAbotURL() + "/api/plugins/train.json",
				method: "DELETE",
				data: {
					Sentence: s.Sentence,
					PluginID: ctrl.props.pluginIdx(),
				},
			}).then(null, function(err) {
				console.error(err)
			})
		}
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
			m.route(window.location.pathname + window.location.hash)
		}
	}
	ctrl.props = {
		title: m.prop("Train"),
		pluginIdx: m.prop(0),
		plugins: m.prop([]),
		sentences: m.prop([]),
		published: m.prop(false),
		error: m.prop(""),
		success: m.prop(""),
	}
	ctrl.subviews = {
		newSentence: m.component(abot.TrainingNewSentence, ctrl),
	}
	abot.Plugins.fetch().then(function(resp) {
		resp.Plugins = resp.Plugins || []

		// Plugin ordering is inconsistent. This sort fixes that.
		resp.Plugins.sort(function(a,b) {
			return (a.Name > b.Name) ? 1 : ((b.Name > a.Name) ? -1 : 0);
		});
		ctrl.props.plugins(resp.Plugins)
		ctrl.selectPlugin(0)
		ctrl.getSentences()
	}, function(err) {
		console.error(err)
	})
}
abot.Training.view = function(ctrl) {
	return m(".container", [
		m.component(abot.Header),
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
							m("h3.top-el", "Sentences"),
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
							value: "Discard",
						}),
						m("input[type=button].btn.btn-primary", {
							onclick: ctrl.save,
							value: "Save",
						}),
					])
				}(),
			]),
		]),
	])
}
})(!window.abot ? window.abot={} : window.abot);
