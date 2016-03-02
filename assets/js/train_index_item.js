(function(abot) {
abot.TrainIndexItem = {}
abot.TrainIndexItem.controller = function(props, pctrl) {
	var ctrl = this
	abot.fnCopy(pctrl, ctrl, "route")
}
abot.TrainIndexItem.view = function(ctrl, props) {
	var t = abot.prettyDate(props.CreatedAt)
	return m("tr", {
		"data-id": props.ID,
		"data-user-id": props.UserID,
		key: props.ID,
		onclick: ctrl.route
	}, [
		m("td", props.Sentence),
		m("td.right", abot.prettyDate(props.CreatedAt))
	])
}
})(!window.abot ? window.abot={} : window.abot);
