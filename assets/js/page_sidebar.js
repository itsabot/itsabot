(function(abot) {
abot.PageSidebar = {}
abot.PageSidebar.view = function(pctrl, _, partial) {
	return m("div", [
		m.component(abot.Header),
		partial(pctrl),
	])
}
})(!window.abot ? window.abot={} : window.abot);
