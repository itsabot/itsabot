(function(ava) {
ava.EarlyAccess = {}
ava.EarlyAccess.controller = function() {
	var ctrl = this
	ctrl.init = function() {
		m.render(document.querySelector("body"), ava.Header.view())
		m.render(document.getElementById("content"), ava.EarlyAccess.view())
	}
}
ava.EarlyAccess.view = function() {
	return m("div", "not implemented")
}
})(!window.ava ? window.ava={} : window.ava);
