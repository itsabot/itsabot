var auth2;
function start() {
	gapi.load("auth2", function() {
		auth2 = gapi.auth2.init({
			client_id: "706975164052-s1tu2v5p2f7vnioee45bh3qbonkfe8qh.apps.googleusercontent.com",
			scope: "https://www.googleapis.com/auth/calendar"
		});
	});
}
function signInCallback(authResult) {
	if (authResult["code"]) {
		console.log(authResult["code"]);
		m.request({
			method: "POST",
			url: window.location.origin + "/oauth/connect/gcal.json",
			data: {
				Code: authResult["code"],
				UserID: parseInt(cookie.getItem("id")),
			},
		}).then(function() {
			console.log("success");
		}, function(err) {
			console.error("err here");
			console.error(err);
		});
	} else {
		console.error("something went wrong");
	}
}

