var earlyAccess = {};
earlyAccess.controller = {
	init: function() {
		m.render(document.querySelector("body"), header.view());
		m.render(document.getElementById("content"), earlyAccess.view());
	}
};
earlyAccess.view = function() {
	return m("div", [

	]);
};
