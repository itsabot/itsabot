(function(ava) {
ava.Index = {}
ava.Index.view = function() {
	return m("div", [
		m.component(ava.Header)
	])
}
})(!window.ava ? window.ava={} : window.ava);
