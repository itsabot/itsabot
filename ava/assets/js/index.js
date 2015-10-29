var Index = {};
Index.controller = function() {
	return {};
};
Index.vm = {
	showEarlyAccess: function() {
		document.getElementById("btns").classList.add("hidden");
		document.getElementById("earlyAccess").classList.remove("hidden");
		setTimeout(function() {
			document.getElementById("earlyAccess").classList.add("fade-in");
		}, 300);
	}
};
Index.view = function() {
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
							href: "/",
							config: m.route
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
								href: "/",
								config: m.route
							}, "Home"),
							m("a", {
								href: "/tour",
								config: m.route
							}, "Tour"),
							m("a", {
								href: "https://medium.com/ava-updates/latest"
							}, "Updates")
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
							m("p", {
								id: "earlyAccess",
								class: "hidden fade"
							}, [
								'Get early access:',
								m("h3", 'Text Ava at ', [
									m("strong", {
										class: "phone"
									}, "(424) 297-1568"),
									' and say "Hi!"'
								])
							]),
							m("div", {
								id: "btns"
							}, [
								m("p", "Get early access to the world's most advanced digital assistant."),
								m("a", {
									class: "btn",
									href: "/tour",
									config: m.route
								}, "Take a tour"),
								m("a", {
									class: "btn btn-green",
									onclick: Index.vm.showEarlyAccess
								}, "Get early access")
							])
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
							href: "https://medium.com/@egtann/car-mechanic-1af70923eb19#.o7htx32u7"
						}, m.trust("Read more &nbsp; &#9654;"))
					])
				])
			])
		])
	]);
};

/*
window.onload = function() {
	Index.controller.init();
};
*/
