var index = {};
index.controller = {
	init: function() {
		m.render(document.querySelector("body"), index.view(index.controller));
	}
};

index.view = function(controller) {
	return m("div", [
		m("div", {
			class: "gradient gradient-big gradient-bright"
		}, [
			m("div", {
				class: "row row-jumbo"
			}, [
				m("header", [
					m("div", {
						class: "container"
					}, [
						m("a", {
							class: "navbar-brand",
							href: "/"
						}, [
							m("div", [
								m("img", {
									src: "/public/images/logo.svg"
								})
							])
						]),
						m("div", {
							class: "text-right navbar-right"
						}, [
							m("a", {
								href: "#"
							}, "Tour"),
							m("a", {
								href: "#"
							}, "Updates"),
							m("a", {
								href: "#"
							}, "Ava for Business"),
							m("a", {
								href: "#"
							}, "About us")
						])
					])
				]),
				m("div", {
					class: "container"
				}, [
					m("div", {
						class: "jumbo row"
					}, [
						m("div", {
							class: "col-md-8"
						}, [
							m("h1", "Meet Ava."),
							m("br"),
							m("h1", "Your new assistant."),
							m("p", "Get early access to the world's most advanced digital assistant."),
							m("a", {
								class: "btn"
							}, "Take a tour"),
							m("a", {
								class: "btn btn-green"
							}, "Get early access")
						]),
						m("div", {
							class: "col-md-4"
						}, [
							m("img", {
								class: "img-big",
								src: "/public/images/iphone.png"
							})
						])
					])
				])
			]),
			m("div", {
				class: "container"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-1"
					}, [
						m("div", {
							class: "label label-primary"
						}, "New")
					]),
					m("div", {
						class: "col-md-4"
					}, [
						m("p", "Car trouble? Ava now finds recommended mechanic and tow services nearby.")
					]),
					m("div", {
						class: "col-md-2"
					}, [
						m("a", {
							class: "bold",
							href: "#"
						}, m.trust("Read more &nbsp; &#9654;"))
					])
				])
			])
		])
	]);
}

window.onload = function() {
	index.controller.init();
};
