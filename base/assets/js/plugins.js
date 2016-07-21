(function(abot) {
abot.Plugins = {}
abot.Plugins.fetch = function() {
	return abot.request({
		method: "GET",
		url: "/api/admin/plugins.json",
	})
}
})(!window.abot ? window.abot={} : window.abot);
