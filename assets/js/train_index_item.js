(function(ava) {
ava.TrainIndexItem = {}
ava.TrainIndexItem.controller = function(props, pctrl) {
	var ctrl = this
	ava.fnCopy(pctrl, ctrl, "route")
}
ava.TrainIndexItem.view = function(ctrl, props) {
	var t = prettyDate(props.CreatedAt)
	return m("tr", {
		"data-id": props.ID,
		"data-user-id": props.UserID,
		key: props.ID,
		onclick: ctrl.route
	}, [
		m("td", props.Sentence),
		m("td", prettyDate(props.CreatedAt))
	])
}
})(!window.ava ? window.ava={} : window.ava);
