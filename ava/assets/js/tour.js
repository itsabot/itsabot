var Tour = {};
Tour.controller = function() {
	return {};
};
Tour.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		m("div", {
			class: "container"
		}, [
			m("div", {
				class: "row"
			}, [
				m("div", {
					class: "col-md-12 margin-top-sm text-center"
				}, [
					m("h1", "She only does everything"),
					m("p", "From scheduling meetings to roadside assistance, Ava’s there for you.")
				])
			]),
			m("div", {
				class: "row"
			}, [
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_burger.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "Eat well"),
					m("p", "Ava finds great restaurants nearby and delivers good eats from any of them."),
				]),
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_martini.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "Enjoy the night"),
					m("p", "Ava knows the best clubs. And she’ll get you in the door for less. "),
				]),
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_medical.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "See the doctor"),
					m("p", "Ava will find you the best care compatible with your health insurance. Many come to you."),
				]),
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_speaker.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "Hear it first"),
					m("p", "Ava’s always on the lookout for great, undiscovered songs. If she hears something you’ll love, she’ll send it to you."),
				]),
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_palm.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "Take a vacation"),
					m("p", "Want to get away? Ava books travel, finding the best value within your budget."),
				]),
				m("div", {
					class: "col-md-1 margin-top-sm"
				}, [
					m("img", {
						src: "/public/images/icon_calendar.svg",
						class: "tour-icon"
					})
				]),
				m("div", {
					class: "col-md-11 margin-top-sm"
				}, [
					m("h4", "Schedule anything"),
					m("p", "Your calendar should be automatic. Let Ava book your appointments and juggle meeting times."),
				])
			]),
			m("div", {
				class: "row margin-top-sm"
			}, [
				m("p", {
					style: "font-style:italic"
				}, "And so much more...")
			]),
			m("div", {
				class: "row margin-top-sm text-center"
			}, [
				m("h2", [
					"Text Ava at ",
					m("span", {
						class: "color-primary"
					}, "(424) 297-1568"),
					' and say, "Hi!"'
				])
			])
		]),
		Footer.view()
	]);
};
