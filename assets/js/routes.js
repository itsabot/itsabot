m.route.mode = "search";

window.onload = function() {
	m.route(document.body, "/", {
		"/": Index,
		"/tour": Tour,
		"/train": Train,
		"/train/:sentenceID": Train,
		"/signup": Signup,
		"/login": Login,
		"/profile": Profile,
		"/cards/new": Card
	});
};
