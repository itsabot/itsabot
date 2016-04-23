(function(abot) {
abot.Phones = {}
abot.Phones.controller = function() {
	var ctrl = this
	ctrl.format = function(number) {
		var a1 = number.slice(0, 2);
		var a2 = " (" + number.slice(2, 5) + ") ";
		var a3 = number.slice(5, 8) + "-"
		return a1 + a2 + a3 + number.slice(8)
	}
}
abot.Phones.view = function(ctrl, props) {
	return m('div', [
		m("h3", "Phone numbers"),
		m("table", [
			m("thead", m("tr", m("th", "Number"))),
			m("tbody", props.map(function(phone) {
				var fmtd = ctrl.format(phone.Number)
				return m.component(abot.Phone, {
					ID: phone.ID,
					Number: fmtd
				})
			}))
		])
	])
}
})(!window.abot ? window.abot={} : window.abot);
