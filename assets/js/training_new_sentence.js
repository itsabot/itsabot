(function(abot) {
abot.TrainingNewSentence = {}
abot.TrainingNewSentence.controller = function(pctrl) {
	var ctrl = this
	ctrl.save = function() {
		var ps = pctrl.props.sentences()
		var s = document.getElementById("sentence").value
		var dupe = false
		console.log(s)
		for (var i = 0; i < ps.length; i++) {
			console.log("vs " + ps[i])
			if (s === ps[i]) {
				dupe = true
				break
			}
		}
		if (dupe) {
			pctrl.props.error("That sentence already exists.")
			return
		}
		abot.externalRequest({
			url: abot.itsAbotURL() + "/api/plugins/train.json",
			method: "POST",
			remotePluginID: pctrl.props.plugins()[pctrl.props.pluginIdx()].ID,
			data: {
				Sentence: s,
				Intent: document.getElementById("intent").value,
			},
		}).then(function(resp) {
			pctrl.props.error("")
			pctrl.props.success("Success! Added sentence.")
			pctrl.props.sentences().unshift(resp)
			ctrl.hide()
		}, function(err) {
			pctrl.props.error(err.Msg)
		})
	}
	ctrl.hide = function() {
		document.getElementById("train-sentence-btn").classList.remove("hidden")
		document.getElementById("train-sentence").classList.add("hidden")
		document.getElementById("sentence").value = ""
		document.getElementById("intent").value = ""
	}
}
abot.TrainingNewSentence.view = function(ctrl) {
	return m("#train-sentence.well.hidden", [
		m("div", [
			m("input[type=text]#sentence.input-clear.input-full", {
				placeholder: "Your sentence here...",
				config: function(el) { el.focus() },
			}),
		]),
		m(".badge-container", [
			m(".badge", "Intent"),
			m("input[type=text]#intent.input-clear.input-sm", {
				placeholder: "set_alarm",
			}),
		]),
		m(".btn-container-right", [
			m("input[type=button].btn", {
				onclick: ctrl.hide,
				value: "Cancel"
			}),
			m("input[type=button].btn.btn-primary", {
				onclick: ctrl.save,
				value: "Save",
			}),
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
