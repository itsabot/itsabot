m.route.mode = "search";

window.onload = function() {
	m.route(document.body, "/", {
		"/": Index,
		"/tour": Tour,
		"/train": TrainIndex,
		"/train/:id": TrainIndexShow
		//"/train": Train,
		//"/train/:sentenceID": Train,
		"/signup": Signup,
		"/login": Login,
		"/forgot_password": ForgotPassword,
		"/reset_password": ResetPassword,
		"/profile": Profile,
		"/cards/new": Card,
	});
};
