m.route.mode = "search";

window.onload = function() {
	m.route(document.body, "/", {
		"/": Index,
		"/tour": Tour,
		"/train": Train
	});
};
