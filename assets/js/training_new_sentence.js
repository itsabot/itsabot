(function(abot) {
abot.TrainingNewSentence = {}
abot.TrainingNewSentence.controller = function(pctrl) {
	var ctrl = this
	ctrl.save = function() {
		m.request({
			url: abot.itsAbotURL() + "/api/plugins/train.json",
			method: "POST",
			data: {
				PluginName: pctrl.props.plugins()[pctrl.props.pluginIdx()].Name,
				Sentence: document.getElementById("sentence").value,
				Intent: document.getElementById("intent").value,
			},
		}).then(function(resp) {
			pctrl.props.error("")
			pctrl.props.success("Added sentence")
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
			m("button.btn", { onclick: ctrl.hide }, "Cancel"),
			m("button.btn.btn-primary", { onclick: ctrl.save }, "Save"),
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
