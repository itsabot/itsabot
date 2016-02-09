var auth2;
function start() {
	gapi.load("auth2", function() {
		auth2 = gapi.auth2.getAuthInstance();
		if (auth2 === null) {
			gapi.auth2.init({
				client_id: "706975164052-s1tu2v5p2f7vnioee45bh3qbonkfe8qh.apps.googleusercontent.com",
				scope: "https://www.googleapis.com/auth/calendar"
			}).then(function(a) {
				auth2 = a;
				if (auth2.isSignedIn.get()) {
					var email = auth2.currentUser.get().getBasicProfile().
						getEmail();
					Profile.vm.toggleGoogleAccount(email);
				}
			}, function(err) {
				console.log("err here");
			});
		}
	});
}
