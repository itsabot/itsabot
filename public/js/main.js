var Card = function(data) {
	var _this = this;
	data = data || {};
	_this.id = m.prop(data.id || 0);
	_this.cardholderName = m.prop(data.cardholderName || "");
	_this.number = m.prop(data.number || "");
	_this.zip5 = m.prop(data.zip5 || "");
	_this.brand = m.prop("");
	if (data.expMonth != null && data.expYear != null) {
		_this.expiry = m.prop(data.expMonth + " / " + data.expYear);
	} else {
		_this.expiry = m.prop(data.expiry || "");
	}
	_this.cvc = m.prop(data.cvc || "");
	_this.last4 = m.prop(data.last4 || "");
	_this.save = function() {
		var deferred = m.deferred();
		saveStripe().then(function(resp) {
			_this.brand(resp.card.brand);
			var data = {
				UserID: parseInt(cookie.getItem("id")),
				StripeToken: resp.id,
				CardholderName: resp.card.name,
				ExpMonth: resp.card.exp_month,
				ExpYear: resp.card.exp_year,
				Brand: _this.brand(),
				Last4: resp.card.last4,
				AddressZip: _this.zip5()
			};
			m.request({
				method: "POST",
				url: "/api/cards.json",
				data: data
			}).then(function(data) {
				deferred.resolve(data);
			}, function(err) {
				deferred.reject(new Error(err.Msg));
			});
		}, function(err) {
			deferred.reject(err);
		});
		return deferred.promise;
	};
	var saveStripe = function() {
		var deferred = m.deferred();
		Stripe.card.createToken({
			number: _this.number(),
			cvc: _this.cvc(),
			exp: _this.expiry(),
			name: _this.cardholderName(),
			address_zip: _this.zip5()
		}, function(status, response) {
			if (response.error) {
				return deferred.reject(new Error(response.error.message));
			}
			deferred.resolve(response);
		});
		return deferred.promise;
	};
};

Card.brandIcon = function(brand) {
	var icon;
	console.log("brand: " + brand);
	switch(brand) {
	case "American Express", "Diners", "Discover", "JCB", "Maestro",
		"MasterCard", "PayPal", "Visa":
		var imgPath = brand.toLowerCase().replace(" ", "_");
		imgPath = "card_" + imgPath + ".svg";
		imgPath = "/public/images/" + imgPath;
		icon = m("img", { src: imgPath, class: "icon-fit" });
		break;
	default:
		icon = m("span", brand);
		break;
	}
	return icon;
};
Card.controller = function() {
	var _this = this;
	_this.vm = new Card.vm(_this);
	_this.card = new Card({});
	_this.error = m.prop("");
	_this.saveCard = function(ev) {
		ev.preventDefault();
		if (_this.vm.saving()) {
			return;
		}
		_this.vm.save();
		_this.error(_this.vm.validateFields());
		if (_this.error() !== "") {
			_this.vm.saveComplete();
			return;
		}
		_this.card.save().then(function(data) {
			m.route("/profile");
			m.redraw();
		}, function(err) {
			_this.error(err.message);
			_this.vm.saveComplete();
		});
	};
};
Card.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Card.addView(controller),
		Footer.view()
	]);
};

Card.addView = function(controller) {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-12"
			}, [
				m("h1", "Add Card")
			])
		]),
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("form", {
				class: "col-md-7 card"
			}, [
				m("div", {
					id: "card-error",
					class: "alert alert-danger hidden"
				}, controller.error()),
				m("div", {
					class: "form-horizontal"
				}, [
					m("div", {
						id: "card-number",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Card number"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "4444 0000 0000 1234",
								onchange: m.withAttr("value", controller.card.number),
								value: controller.card.number()
							})
						])
					]),
					m("div", {
						id: "card-expiry",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Expires"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "01 / 2015",
								onchange: m.withAttr("value", controller.card.expiry),
								value: controller.card.expiry()
							})
						])
					]),
					m("div", {
						id: "card-cvc",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "CVC"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "123",
								onchange: m.withAttr("value", controller.card.cvc),
								value: controller.card.cvc()
							})
						])
					]),
					m("div", {
						id: "card-name",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Cardholder name"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "Cardholder name",
								onchange: m.withAttr("value", controller.card.cardholderName),
								value: controller.card.cardholderName()
							})
						])
					]),
					m("div", {
						id: "card-name",
						class: "form-group"
					}, [
						m("label", {
							class: "col-md-3 control-label"
						}, "Billing zip"),
						m("div", {
							class: "col-md-9"
						}, [
							m("input", {
								class: "form-control",
								type: "text",
								placeholder: "90210",
								onchange: m.withAttr("value", controller.card.zip5),
								value: controller.card.zip5()
							})
						])
					])
				]),
				m("div", {
					class: "text-right"
				}, [
					m("a", {
						id: "card-cancel-btn",
						class: "btn btn-sm",
						href: "/profile",
						config: m.route
					}, "Cancel"),
					m("input", {
						id: "card-save-btn",
						type: "submit",
						class: "btn btn-primary btn-sm btn-collection",
						value: controller.vm.savingText(),
						onclick: controller.saveCard,
						onsubmit: controller.saveCard
					})
				])
			])
		])
	]);
};

Card.listView = function(list) {
	return m("table", {
		class: "table"
	}, [
		m("thead", [
			m("tr", [
				m("th", "Type"),
				m("th", "Cardholder Name"),
				m("th", "Number"),
				m("th", "Expires"),
			])
		]),
		m("tbody", [
			list.data().map(function(item) {
				return m("tr", {
					"key": item.Id
				}, [
					m("td", {
						style: "width: 10%"
					}, Card.brandIcon(item.Brand)),
					m("td", item.CardholderName),
					m("td", {class: "subtle"}, "XXXX-" + item.Last4),
					m("td", item.ExpMonth + " / " + item.ExpYear),
					m("td", {
						class: "text-right"
					}, [
						m("img", {
							class: "icon icon-xs icon-delete",
							src: "/public/images/icon_delete.svg",
							onclick: function() {
								var c = confirm("Delete this number?");
								if (c) {
									// TODO delete from database and update view
									console.warn("not implemented");
								}
							}
						})
					])
				])
			})
		])
	]);
};
Card.vm = function(controller) {
	var saveBtn = function() {
		return document.getElementById("card-save-btn");
	};
	var cancelBtn = function() {
		return document.getElementById("card-cancel-btn");
	};
	var errorHolder = function() {
		return document.getElementById("card-error");
	};
	var cardNumberHolder = function() {
		return document.getElementById("card-number");
	}
	var cardExpiryHolder = function() {
		return document.getElementById("card-expiry");
	};
	var cardCVCHolder = function() {
		return document.getElementById("card-cvc");
	};
	var _this = this;
	_this.saving = m.prop(false);
	_this.savingText = m.prop("Save");
	_this.save = function() {
		_this.saving(true);
		_this.savingText("Saving...");
		cancelBtn().classList.add("hidden");
		errorHolder().classList.add("hidden");
	};
	_this.saveComplete = function() {
		_this.saving(false);
		cancelBtn().classList.remove("hidden");
		_this.savingText("Save");
		if (controller.error() != null && controller.error() !== "") {
			errorHolder().innerText = controller.error();
			errorHolder().classList.remove("hidden");
		} else {
			errorHolder().classList.add("hidden");
		}
	};
	_this.validateFields = function() {
		var card = controller.card;
		if (Stripe.card.validateCardNumber(card.number())) {
			cardNumberHolder().classList.remove("has-error");
		} else {
			cardNumberHolder().classList.add("has-error");
			return "Card number is invalid.";
		}
		if (Stripe.card.validateExpiry(card.expiry())) {
			cardExpiryHolder().classList.remove("has-error");
		} else {
			cardExpiryHolder().classList.add("has-error");
			return "Card expiration is invalid.";
		}
		if (Stripe.card.validateCVC(card.cvc())) {
			cardCVCHolder().classList.remove("has-error");
		} else {
			cardCVCHolder().classList.add("has-error");
			return "Card CVC is invalid. This is the 3 or 4 digit security code.";
		}
		return "";
	};
};
/*\
 * |*|
 * |*|  :: cookies.js ::
 * |*|
 * |*|  A complete cookies reader/writer framework with full unicode support.
 * |*|
 * |*|  Revision #1 - September 4, 2014
 * |*|
 * |*|  https://developer.mozilla.org/en-US/docs/Web/API/document.cookie
 * |*|  https://developer.mozilla.org/User:fusionchess
 * |*|
 * |*|  This framework is released under the GNU Public License, version 3 or later.
 * |*|  http://www.gnu.org/licenses/gpl-3.0-standalone.html
 * |*|
 * |*|  Syntaxes:
 * |*|
 * |*|  * cookie.setItem(name, value[, end[, path[, domain[, secure]]]])
 * |*|  * cookie.getItem(name)
 * |*|  * cookie.removeItem(name[, path[, domain]])
 * |*|  * cookie.hasItem(name)
 * |*|  * cookie.keys()
 * |*|
 * \*/

var cookie = {
	getItem: function(sKey) {
		if (!sKey) {
			return null;
		}
		return decodeURIComponent(document.cookie.replace(new RegExp("(?:(?:^|.*;)\\s*" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=\\s*([^;]*).*$)|^.*$"), "$1")) || null;
	},
	setItem: function(sKey, sValue, vEnd, sPath, sDomain, bSecure) {
		if (!sKey || /^(?:expires|max\-age|path|domain|secure)$/i.test(sKey)) {
			return false;
		}
		var sExpires = "";
		if (vEnd) {
			switch (vEnd.constructor) {
				case Number:
					sExpires = vEnd === Infinity ? "; expires=Fri, 31 Dec 9999 23:59:59 GMT" : "; max-age=" + vEnd;
					break;
				case String:
					sExpires = "; expires=" + vEnd;
					break;
				case Date:
					sExpires = "; expires=" + vEnd.toUTCString();
					break;
			}
		}
		document.cookie = encodeURIComponent(sKey) + "=" + encodeURIComponent(sValue) + sExpires + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "") + (bSecure ? "; secure" : "");
		return true;
	},
	removeItem: function(sKey, sPath, sDomain) {
		if (!this.hasItem(sKey)) {
			return false;
		}
		document.cookie = encodeURIComponent(sKey) + "=; expires=Thu, 01 Jan 1970 00:00:00 GMT" + (sDomain ? "; domain=" + sDomain : "") + (sPath ? "; path=" + sPath : "");
		return true;
	},
	hasItem: function(sKey) {
		if (!sKey) {
			return false;
		}
		return (new RegExp("(?:^|;\\s*)" + encodeURIComponent(sKey).replace(/[\-\.\+\*]/g, "\\$&") + "\\s*\\=")).test(document.cookie);
	},
	keys: function() {
		var aKeys = document.cookie.replace(/((?:^|\s*;)[^\=]+)(?=;|$)|^\s*|\s*(?:\=[^;]*)?(?:\1|$)/g, "").split(/\s*(?:\=[^;]*)?;\s*/);
		for (var nLen = aKeys.length, nIdx = 0; nIdx < nLen; nIdx++) {
			aKeys[nIdx] = decodeURIComponent(aKeys[nIdx]);
		}
		return aKeys;
	}
};
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
var Footer = {
	controller: function() {
		return {};
	},
	view: function() {
		return m("footer", [
			m("div", {
				class: "container"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-3 border-right"
					}, [
						m("div", {
							class: "big-name"
						}, "Ava"),
						m("div", {
							class: "big-name big-name-gray"
						}, "Assistant"),
						m("div", {
							class: "margin-top-sm"
						}, m.trust("&copy; 2015 Evan Tann.")),
						m("div", "All rights reserved.")
					]),
					m("div", {
						class: "col-md-2"
					}, [
						m("div", m("a", {
							href: "/",
							config: m.route
						}, "Food")),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Travel")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Health")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Shopping")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/",
								config: m.route
							}, "Entertainment")
						])
					]),
					m("div", {
						class: "col-md-7 de-emphasized"
					}, [
						m("div", [
							m("a", {
								href: "/",
								config: m.route
							}, "Home")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/tour",
								config: m.route
							}, "Tour")
						]),
						m("div", {
							class: "margin-top-xs"
						}, [
							m("a", {
								href: "/updates",
								config: m.route
							}, "Updates")
						]),
					])
				])
			])
		]);
	}
};
var header = {};
header.view = function() {
	return m("header", {
		class: "gradient"
	}, [
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
					}),
					m("span", {
						class: "margin-top-xs"
					}, m.trust(" &nbsp;Ava")),
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
				}, "Updates"),
				function() {
					if (cookie.getItem("id") !== null) {
						return m("a", {
							href: "/profile",
							config: m.route
						}, "Profile")
					}
					return m("a", {
						href: "/login",
						config: m.route
					}, "Log in")
				}()
			])
		]),
		m("div", {
			id: "content"
		})
	]);
};
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
				class: "jumbo-container"
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
							}, "Updates"),
							function() {
								if (cookie.getItem("id") !== null) {
									return m("a", {
										href: "/profile",
										config: m.route
									}, "Profile")
								}
								return m("a", {
									href: "/login",
									config: m.route
								}, "Log in")
							}()
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
var List = function(data) {
	var _this = this;
	_this.id = Math.floor((Math.random() * 1000000000) + 1);
	_this.userId = m.prop(cookie.getItem("id"));
	_this.type = m.prop(data.type);
	_this.placeholder = m.prop(data.placeholder || "");
	_this.data = m.prop([]);
	_this.view = function() {
		return m("div", {class: "table-responsive"}, [
			function() {
				return _this.type().listView(_this);
			}()
		]);
	};
};
var Login = {
	login: function(ev) {
		ev.preventDefault();
		var email = document.getElementById("email").value;
		var pass = document.getElementById("password").value;
		return m.request({
			method: "POST",
			data: {
				email: email,
				password: pass
			},
			url: "/api/login.json"
		}).then(function(data) {
			var date = new Date();
			var exp = date.setDate(date + 30);
			var secure = true;
			if (window.location.hostname === "localhost") {
				secure = false;
			}
			cookie.setItem("id", data.Id, exp, null, null, secure);
			cookie.setItem("session_token", data.SessionToken, exp, null, null, secure);
			m.route("/profile");
		}, function(err) {
			Login.controller.error(err.Msg);
		});
	},
	checkAuth: function(callback) {
		if (cookie.getItem("id") !== null) {
			callback(true);
		}
	}
};

Login.controller = function() {
	Login.checkAuth(function(loggedIn) {
		if (loggedIn) {
			return m.route("/profile");
		}
	});
	Login.controller.error = m.prop("");
};

Login.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Login.viewFull(),
		Footer.view()
	]);
}

Login.viewFull = function() {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-push-3 col-md-6 card"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("h2", "Log In")
					])
				]),
				m("form", [
					m("div", {
						class: "row margin-top-sm"
					}, [
						m("div", {
							class: "col-md-12"
						}, [

							function() {
								if (Login.controller.error() !== "") {
									return m("div", {
										class: "alert alert-danger"
									}, Login.controller.error());
								}
							}(),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "email",
									class: "form-control",
									id: "email",
									placeholder: "Email"
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "password",
									class: "form-control",
									id: "password",
									placeholder: "Password"
								})
							]),
							m("div", {
								class: "form-group text-right"
							}, [
								m("a", "Forgot password?")
							])
						])
					]),
					m("div", {
						class: "row"
					}, [
						m("div", {
							class: "col-md-12 text-center"
						}, [
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									class: "btn btn-sm",
									id: "btn",
									type: "submit",
									onclick: Login.login,
									onsubmit: Login.login,
									value: "Log In"
								})
							])
						])
					])
				]),
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("div", {
							class: "form-group"
						}, [
							m("span", "No account? "),
							m("a", {
								href: "/signup",
								config: m.route
							}, "Sign Up")
						])
					])
				])
			])
		])
	]);
};
var m = (function app(window, undefined) {
	var OBJECT = "[object Object]", ARRAY = "[object Array]", STRING = "[object String]", FUNCTION = "function";
	var type = {}.toString;
	var parser = /(?:(^|#|\.)([^#\.\[\]]+))|(\[.+?\])/g, attrParser = /\[(.+?)(?:=("|'|)(.*?)\2)?\]/;
	var voidElements = /^(AREA|BASE|BR|COL|COMMAND|EMBED|HR|IMG|INPUT|KEYGEN|LINK|META|PARAM|SOURCE|TRACK|WBR)$/;
	var noop = function() {}

	// caching commonly used variables
	var $document, $location, $requestAnimationFrame, $cancelAnimationFrame;

	// self invoking function needed because of the way mocks work
	function initialize(window){
		$document = window.document;
		$location = window.location;
		$cancelAnimationFrame = window.cancelAnimationFrame || window.clearTimeout;
		$requestAnimationFrame = window.requestAnimationFrame || window.setTimeout;
	}

	initialize(window);


	/**
	 * @typedef {String} Tag
	 * A string that looks like -> div.classname#id[param=one][param2=two]
	 * Which describes a DOM node
	 */

	/**
	 *
	 * @param {Tag} The DOM node tag
	 * @param {Object=[]} optional key-value pairs to be mapped to DOM attrs
	 * @param {...mNode=[]} Zero or more Mithril child nodes. Can be an array, or splat (optional)
	 *
	 */
	function m() {
		var args = [].slice.call(arguments);
		var hasAttrs = args[1] != null && type.call(args[1]) === OBJECT && !("tag" in args[1] || "view" in args[1]) && !("subtree" in args[1]);
		var attrs = hasAttrs ? args[1] : {};
		var classAttrName = "class" in attrs ? "class" : "className";
		var cell = {tag: "div", attrs: {}};
		var match, classes = [];
		if (type.call(args[0]) != STRING) throw new Error("selector in m(selector, attrs, children) should be a string")
		while (match = parser.exec(args[0])) {
			if (match[1] === "" && match[2]) cell.tag = match[2];
			else if (match[1] === "#") cell.attrs.id = match[2];
			else if (match[1] === ".") classes.push(match[2]);
			else if (match[3][0] === "[") {
				var pair = attrParser.exec(match[3]);
				cell.attrs[pair[1]] = pair[3] || (pair[2] ? "" :true)
			}
		}

		var children = hasAttrs ? args.slice(2) : args.slice(1);
		if (children.length === 1 && type.call(children[0]) === ARRAY) {
			cell.children = children[0]
		}
		else {
			cell.children = children
		}
		
		for (var attrName in attrs) {
			if (attrs.hasOwnProperty(attrName)) {
				if (attrName === classAttrName && attrs[attrName] != null && attrs[attrName] !== "") {
					classes.push(attrs[attrName])
					cell.attrs[attrName] = "" //create key in correct iteration order
				}
				else cell.attrs[attrName] = attrs[attrName]
			}
		}
		if (classes.length > 0) cell.attrs[classAttrName] = classes.join(" ");
		
		return cell
	}
	function build(parentElement, parentTag, parentCache, parentIndex, data, cached, shouldReattach, index, editable, namespace, configs) {
		//`build` is a recursive function that manages creation/diffing/removal of DOM elements based on comparison between `data` and `cached`
		//the diff algorithm can be summarized as this:
		//1 - compare `data` and `cached`
		//2 - if they are different, copy `data` to `cached` and update the DOM based on what the difference is
		//3 - recursively apply this algorithm for every array and for the children of every virtual element

		//the `cached` data structure is essentially the same as the previous redraw's `data` data structure, with a few additions:
		//- `cached` always has a property called `nodes`, which is a list of DOM elements that correspond to the data represented by the respective virtual element
		//- in order to support attaching `nodes` as a property of `cached`, `cached` is *always* a non-primitive object, i.e. if the data was a string, then cached is a String instance. If data was `null` or `undefined`, cached is `new String("")`
		//- `cached also has a `configContext` property, which is the state storage object exposed by config(element, isInitialized, context)
		//- when `cached` is an Object, it represents a virtual element; when it's an Array, it represents a list of elements; when it's a String, Number or Boolean, it represents a text node

		//`parentElement` is a DOM element used for W3C DOM API calls
		//`parentTag` is only used for handling a corner case for textarea values
		//`parentCache` is used to remove nodes in some multi-node cases
		//`parentIndex` and `index` are used to figure out the offset of nodes. They're artifacts from before arrays started being flattened and are likely refactorable
		//`data` and `cached` are, respectively, the new and old nodes being diffed
		//`shouldReattach` is a flag indicating whether a parent node was recreated (if so, and if this node is reused, then this node must reattach itself to the new parent)
		//`editable` is a flag that indicates whether an ancestor is contenteditable
		//`namespace` indicates the closest HTML namespace as it cascades down from an ancestor
		//`configs` is a list of config functions to run after the topmost `build` call finishes running

		//there's logic that relies on the assumption that null and undefined data are equivalent to empty strings
		//- this prevents lifecycle surprises from procedural helpers that mix implicit and explicit return statements (e.g. function foo() {if (cond) return m("div")}
		//- it simplifies diffing code
		//data.toString() might throw or return null if data is the return value of Console.log in Firefox (behavior depends on version)
		try {if (data == null || data.toString() == null) data = "";} catch (e) {data = ""}
		if (data.subtree === "retain") return cached;
		var cachedType = type.call(cached), dataType = type.call(data);
		if (cached == null || cachedType !== dataType) {
			if (cached != null) {
				if (parentCache && parentCache.nodes) {
					var offset = index - parentIndex;
					var end = offset + (dataType === ARRAY ? data : cached.nodes).length;
					clear(parentCache.nodes.slice(offset, end), parentCache.slice(offset, end))
				}
				else if (cached.nodes) clear(cached.nodes, cached)
			}
			cached = new data.constructor;
			if (cached.tag) cached = {}; //if constructor creates a virtual dom element, use a blank object as the base cached node instead of copying the virtual el (#277)
			cached.nodes = []
		}

		if (dataType === ARRAY) {
			//recursively flatten array
			for (var i = 0, len = data.length; i < len; i++) {
				if (type.call(data[i]) === ARRAY) {
					data = data.concat.apply([], data);
					i-- //check current index again and flatten until there are no more nested arrays at that index
					len = data.length
				}
			}
			
			var nodes = [], intact = cached.length === data.length, subArrayCount = 0;

			//keys algorithm: sort elements without recreating them if keys are present
			//1) create a map of all existing keys, and mark all for deletion
			//2) add new keys to map and mark them for addition
			//3) if key exists in new list, change action from deletion to a move
			//4) for each key, handle its corresponding action as marked in previous steps
			var DELETION = 1, INSERTION = 2 , MOVE = 3;
			var existing = {}, shouldMaintainIdentities = false;
			for (var i = 0; i < cached.length; i++) {
				if (cached[i] && cached[i].attrs && cached[i].attrs.key != null) {
					shouldMaintainIdentities = true;
					existing[cached[i].attrs.key] = {action: DELETION, index: i}
				}
			}
			
			var guid = 0
			for (var i = 0, len = data.length; i < len; i++) {
				if (data[i] && data[i].attrs && data[i].attrs.key != null) {
					for (var j = 0, len = data.length; j < len; j++) {
						if (data[j] && data[j].attrs && data[j].attrs.key == null) data[j].attrs.key = "__mithril__" + guid++
					}
					break
				}
			}
			
			if (shouldMaintainIdentities) {
				var keysDiffer = false
				if (data.length != cached.length) keysDiffer = true
				else for (var i = 0, cachedCell, dataCell; cachedCell = cached[i], dataCell = data[i]; i++) {
					if (cachedCell.attrs && dataCell.attrs && cachedCell.attrs.key != dataCell.attrs.key) {
						keysDiffer = true
						break
					}
				}
				
				if (keysDiffer) {
					for (var i = 0, len = data.length; i < len; i++) {
						if (data[i] && data[i].attrs) {
							if (data[i].attrs.key != null) {
								var key = data[i].attrs.key;
								if (!existing[key]) existing[key] = {action: INSERTION, index: i};
								else existing[key] = {
									action: MOVE,
									index: i,
									from: existing[key].index,
									element: cached.nodes[existing[key].index] || $document.createElement("div")
								}
							}
						}
					}
					var actions = []
					for (var prop in existing) actions.push(existing[prop])
					var changes = actions.sort(sortChanges);
					var newCached = new Array(cached.length)
					newCached.nodes = cached.nodes.slice()

					for (var i = 0, change; change = changes[i]; i++) {
						if (change.action === DELETION) {
							clear(cached[change.index].nodes, cached[change.index]);
							newCached.splice(change.index, 1)
						}
						if (change.action === INSERTION) {
							var dummy = $document.createElement("div");
							dummy.key = data[change.index].attrs.key;
							parentElement.insertBefore(dummy, parentElement.childNodes[change.index] || null);
							newCached.splice(change.index, 0, {attrs: {key: data[change.index].attrs.key}, nodes: [dummy]})
							newCached.nodes[change.index] = dummy
						}

						if (change.action === MOVE) {
							if (parentElement.childNodes[change.index] !== change.element && change.element !== null) {
								parentElement.insertBefore(change.element, parentElement.childNodes[change.index] || null)
							}
							newCached[change.index] = cached[change.from]
							newCached.nodes[change.index] = change.element
						}
					}
					cached = newCached;
				}
			}
			//end key algorithm

			for (var i = 0, cacheCount = 0, len = data.length; i < len; i++) {
				//diff each item in the array
				var item = build(parentElement, parentTag, cached, index, data[i], cached[cacheCount], shouldReattach, index + subArrayCount || subArrayCount, editable, namespace, configs);
				if (item === undefined) continue;
				if (!item.nodes.intact) intact = false;
				if (item.$trusted) {
					//fix offset of next element if item was a trusted string w/ more than one html element
					//the first clause in the regexp matches elements
					//the second clause (after the pipe) matches text nodes
					subArrayCount += (item.match(/<[^\/]|\>\s*[^<]/g) || [0]).length
				}
				else subArrayCount += type.call(item) === ARRAY ? item.length : 1;
				cached[cacheCount++] = item
			}
			if (!intact) {
				//diff the array itself
				
				//update the list of DOM nodes by collecting the nodes from each item
				for (var i = 0, len = data.length; i < len; i++) {
					if (cached[i] != null) nodes.push.apply(nodes, cached[i].nodes)
				}
				//remove items from the end of the array if the new array is shorter than the old one
				//if errors ever happen here, the issue is most likely a bug in the construction of the `cached` data structure somewhere earlier in the program
				for (var i = 0, node; node = cached.nodes[i]; i++) {
					if (node.parentNode != null && nodes.indexOf(node) < 0) clear([node], [cached[i]])
				}
				if (data.length < cached.length) cached.length = data.length;
				cached.nodes = nodes
			}
		}
		else if (data != null && dataType === OBJECT) {
			var views = [], controllers = []
			while (data.view) {
				var view = data.view.$original || data.view
				var controllerIndex = m.redraw.strategy() == "diff" && cached.views ? cached.views.indexOf(view) : -1
				var controller = controllerIndex > -1 ? cached.controllers[controllerIndex] : new (data.controller || noop)
				var key = data && data.attrs && data.attrs.key
				data = pendingRequests == 0 || (cached && cached.controllers && cached.controllers.indexOf(controller) > -1) ? data.view(controller) : {tag: "placeholder"}
				if (data.subtree === "retain") return cached;
				if (key) {
					if (!data.attrs) data.attrs = {}
					data.attrs.key = key
				}
				if (controller.onunload) unloaders.push({controller: controller, handler: controller.onunload})
				views.push(view)
				controllers.push(controller)
			}
			if (!data.tag && controllers.length) throw new Error("Component template must return a virtual element, not an array, string, etc.")
			if (!data.attrs) data.attrs = {};
			if (!cached.attrs) cached.attrs = {};

			var dataAttrKeys = Object.keys(data.attrs)
			var hasKeys = dataAttrKeys.length > ("key" in data.attrs ? 1 : 0)
			//if an element is different enough from the one in cache, recreate it
			if (data.tag != cached.tag || dataAttrKeys.sort().join() != Object.keys(cached.attrs).sort().join() || data.attrs.id != cached.attrs.id || data.attrs.key != cached.attrs.key || (m.redraw.strategy() == "all" && (!cached.configContext || cached.configContext.retain !== true)) || (m.redraw.strategy() == "diff" && cached.configContext && cached.configContext.retain === false)) {
				if (cached.nodes.length) clear(cached.nodes);
				if (cached.configContext && typeof cached.configContext.onunload === FUNCTION) cached.configContext.onunload()
				if (cached.controllers) {
					for (var i = 0, controller; controller = cached.controllers[i]; i++) {
						if (typeof controller.onunload === FUNCTION) controller.onunload({preventDefault: noop})
					}
				}
			}
			if (type.call(data.tag) != STRING) return;

			var node, isNew = cached.nodes.length === 0;
			if (data.attrs.xmlns) namespace = data.attrs.xmlns;
			else if (data.tag === "svg") namespace = "http://www.w3.org/2000/svg";
			else if (data.tag === "math") namespace = "http://www.w3.org/1998/Math/MathML";
			
			if (isNew) {
				if (data.attrs.is) node = namespace === undefined ? $document.createElement(data.tag, data.attrs.is) : $document.createElementNS(namespace, data.tag, data.attrs.is);
				else node = namespace === undefined ? $document.createElement(data.tag) : $document.createElementNS(namespace, data.tag);
				cached = {
					tag: data.tag,
					//set attributes first, then create children
					attrs: hasKeys ? setAttributes(node, data.tag, data.attrs, {}, namespace) : data.attrs,
					children: data.children != null && data.children.length > 0 ?
						build(node, data.tag, undefined, undefined, data.children, cached.children, true, 0, data.attrs.contenteditable ? node : editable, namespace, configs) :
						data.children,
					nodes: [node]
				};
				if (controllers.length) {
					cached.views = views
					cached.controllers = controllers
					for (var i = 0, controller; controller = controllers[i]; i++) {
						if (controller.onunload && controller.onunload.$old) controller.onunload = controller.onunload.$old
						if (pendingRequests && controller.onunload) {
							var onunload = controller.onunload
							controller.onunload = noop
							controller.onunload.$old = onunload
						}
					}
				}
				
				if (cached.children && !cached.children.nodes) cached.children.nodes = [];
				//edge case: setting value on <select> doesn't work before children exist, so set it again after children have been created
				if (data.tag === "select" && "value" in data.attrs) setAttributes(node, data.tag, {value: data.attrs.value}, {}, namespace);
				parentElement.insertBefore(node, parentElement.childNodes[index] || null)
			}
			else {
				node = cached.nodes[0];
				if (hasKeys) setAttributes(node, data.tag, data.attrs, cached.attrs, namespace);
				cached.children = build(node, data.tag, undefined, undefined, data.children, cached.children, false, 0, data.attrs.contenteditable ? node : editable, namespace, configs);
				cached.nodes.intact = true;
				if (controllers.length) {
					cached.views = views
					cached.controllers = controllers
				}
				if (shouldReattach === true && node != null) parentElement.insertBefore(node, parentElement.childNodes[index] || null)
			}
			//schedule configs to be called. They are called after `build` finishes running
			if (typeof data.attrs["config"] === FUNCTION) {
				var context = cached.configContext = cached.configContext || {};

				// bind
				var callback = function(data, args) {
					return function() {
						return data.attrs["config"].apply(data, args)
					}
				};
				configs.push(callback(data, [node, !isNew, context, cached]))
			}
		}
		else if (typeof data != FUNCTION) {
			//handle text nodes
			var nodes;
			if (cached.nodes.length === 0) {
				if (data.$trusted) {
					nodes = injectHTML(parentElement, index, data)
				}
				else {
					nodes = [$document.createTextNode(data)];
					if (!parentElement.nodeName.match(voidElements)) parentElement.insertBefore(nodes[0], parentElement.childNodes[index] || null)
				}
				cached = "string number boolean".indexOf(typeof data) > -1 ? new data.constructor(data) : data;
				cached.nodes = nodes
			}
			else if (cached.valueOf() !== data.valueOf() || shouldReattach === true) {
				nodes = cached.nodes;
				if (!editable || editable !== $document.activeElement) {
					if (data.$trusted) {
						clear(nodes, cached);
						nodes = injectHTML(parentElement, index, data)
					}
					else {
						//corner case: replacing the nodeValue of a text node that is a child of a textarea/contenteditable doesn't work
						//we need to update the value property of the parent textarea or the innerHTML of the contenteditable element instead
						if (parentTag === "textarea") parentElement.value = data;
						else if (editable) editable.innerHTML = data;
						else {
							if (nodes[0].nodeType === 1 || nodes.length > 1) { //was a trusted string
								clear(cached.nodes, cached);
								nodes = [$document.createTextNode(data)]
							}
							parentElement.insertBefore(nodes[0], parentElement.childNodes[index] || null);
							nodes[0].nodeValue = data
						}
					}
				}
				cached = new data.constructor(data);
				cached.nodes = nodes
			}
			else cached.nodes.intact = true
		}

		return cached
	}
	function sortChanges(a, b) {return a.action - b.action || a.index - b.index}
	function setAttributes(node, tag, dataAttrs, cachedAttrs, namespace) {
		for (var attrName in dataAttrs) {
			var dataAttr = dataAttrs[attrName];
			var cachedAttr = cachedAttrs[attrName];
			if (!(attrName in cachedAttrs) || (cachedAttr !== dataAttr)) {
				cachedAttrs[attrName] = dataAttr;
				try {
					//`config` isn't a real attributes, so ignore it
					if (attrName === "config" || attrName == "key") continue;
					//hook event handlers to the auto-redrawing system
					else if (typeof dataAttr === FUNCTION && attrName.indexOf("on") === 0) {
						node[attrName] = autoredraw(dataAttr, node)
					}
					//handle `style: {...}`
					else if (attrName === "style" && dataAttr != null && type.call(dataAttr) === OBJECT) {
						for (var rule in dataAttr) {
							if (cachedAttr == null || cachedAttr[rule] !== dataAttr[rule]) node.style[rule] = dataAttr[rule]
						}
						for (var rule in cachedAttr) {
							if (!(rule in dataAttr)) node.style[rule] = ""
						}
					}
					//handle SVG
					else if (namespace != null) {
						if (attrName === "href") node.setAttributeNS("http://www.w3.org/1999/xlink", "href", dataAttr);
						else if (attrName === "className") node.setAttribute("class", dataAttr);
						else node.setAttribute(attrName, dataAttr)
					}
					//handle cases that are properties (but ignore cases where we should use setAttribute instead)
					//- list and form are typically used as strings, but are DOM element references in js
					//- when using CSS selectors (e.g. `m("[style='']")`), style is used as a string, but it's an object in js
					else if (attrName in node && !(attrName === "list" || attrName === "style" || attrName === "form" || attrName === "type" || attrName === "width" || attrName === "height")) {
						//#348 don't set the value if not needed otherwise cursor placement breaks in Chrome
						if (tag !== "input" || node[attrName] !== dataAttr) node[attrName] = dataAttr
					}
					else node.setAttribute(attrName, dataAttr)
				}
				catch (e) {
					//swallow IE's invalid argument errors to mimic HTML's fallback-to-doing-nothing-on-invalid-attributes behavior
					if (e.message.indexOf("Invalid argument") < 0) throw e
				}
			}
			//#348 dataAttr may not be a string, so use loose comparison (double equal) instead of strict (triple equal)
			else if (attrName === "value" && tag === "input" && node.value != dataAttr) {
				node.value = dataAttr
			}
		}
		return cachedAttrs
	}
	function clear(nodes, cached) {
		for (var i = nodes.length - 1; i > -1; i--) {
			if (nodes[i] && nodes[i].parentNode) {
				try {nodes[i].parentNode.removeChild(nodes[i])}
				catch (e) {} //ignore if this fails due to order of events (see http://stackoverflow.com/questions/21926083/failed-to-execute-removechild-on-node)
				cached = [].concat(cached);
				if (cached[i]) unload(cached[i])
			}
		}
		if (nodes.length != 0) nodes.length = 0
	}
	function unload(cached) {
		if (cached.configContext && typeof cached.configContext.onunload === FUNCTION) {
			cached.configContext.onunload();
			cached.configContext.onunload = null
		}
		if (cached.controllers) {
			for (var i = 0, controller; controller = cached.controllers[i]; i++) {
				if (typeof controller.onunload === FUNCTION) controller.onunload({preventDefault: noop});
			}
		}
		if (cached.children) {
			if (type.call(cached.children) === ARRAY) {
				for (var i = 0, child; child = cached.children[i]; i++) unload(child)
			}
			else if (cached.children.tag) unload(cached.children)
		}
	}
	function injectHTML(parentElement, index, data) {
		var nextSibling = parentElement.childNodes[index];
		if (nextSibling) {
			var isElement = nextSibling.nodeType != 1;
			var placeholder = $document.createElement("span");
			if (isElement) {
				parentElement.insertBefore(placeholder, nextSibling || null);
				placeholder.insertAdjacentHTML("beforebegin", data);
				parentElement.removeChild(placeholder)
			}
			else nextSibling.insertAdjacentHTML("beforebegin", data)
		}
		else parentElement.insertAdjacentHTML("beforeend", data);
		var nodes = [];
		while (parentElement.childNodes[index] !== nextSibling) {
			nodes.push(parentElement.childNodes[index]);
			index++
		}
		return nodes
	}
	function autoredraw(callback, object) {
		return function(e) {
			e = e || event;
			m.redraw.strategy("diff");
			m.startComputation();
			try {return callback.call(object, e)}
			finally {
				endFirstComputation()
			}
		}
	}

	var html;
	var documentNode = {
		appendChild: function(node) {
			if (html === undefined) html = $document.createElement("html");
			if ($document.documentElement && $document.documentElement !== node) {
				$document.replaceChild(node, $document.documentElement)
			}
			else $document.appendChild(node);
			this.childNodes = $document.childNodes
		},
		insertBefore: function(node) {
			this.appendChild(node)
		},
		childNodes: []
	};
	var nodeCache = [], cellCache = {};
	m.render = function(root, cell, forceRecreation) {
		var configs = [];
		if (!root) throw new Error("Ensure the DOM element being passed to m.route/m.mount/m.render is not undefined.");
		var id = getCellCacheKey(root);
		var isDocumentRoot = root === $document;
		var node = isDocumentRoot || root === $document.documentElement ? documentNode : root;
		if (isDocumentRoot && cell.tag != "html") cell = {tag: "html", attrs: {}, children: cell};
		if (cellCache[id] === undefined) clear(node.childNodes);
		if (forceRecreation === true) reset(root);
		cellCache[id] = build(node, null, undefined, undefined, cell, cellCache[id], false, 0, null, undefined, configs);
		for (var i = 0, len = configs.length; i < len; i++) configs[i]()
	};
	function getCellCacheKey(element) {
		var index = nodeCache.indexOf(element);
		return index < 0 ? nodeCache.push(element) - 1 : index
	}

	m.trust = function(value) {
		value = new String(value);
		value.$trusted = true;
		return value
	};

	function gettersetter(store) {
		var prop = function() {
			if (arguments.length) store = arguments[0];
			return store
		};

		prop.toJSON = function() {
			return store
		};

		return prop
	}

	m.prop = function (store) {
		//note: using non-strict equality check here because we're checking if store is null OR undefined
		if (((store != null && type.call(store) === OBJECT) || typeof store === FUNCTION) && typeof store.then === FUNCTION) {
			return propify(store)
		}

		return gettersetter(store)
	};

	var roots = [], components = [], controllers = [], lastRedrawId = null, lastRedrawCallTime = 0, computePreRedrawHook = null, computePostRedrawHook = null, prevented = false, topComponent, unloaders = [];
	var FRAME_BUDGET = 16; //60 frames per second = 1 call per 16 ms
	function parameterize(component, args) {
		var controller = function() {
			return (component.controller || noop).apply(this, args) || this
		}
		var view = function(ctrl) {
			if (arguments.length > 1) args = args.concat([].slice.call(arguments, 1))
			return component.view.apply(component, args ? [ctrl].concat(args) : [ctrl])
		}
		view.$original = component.view
		var output = {controller: controller, view: view}
		if (args[0] && args[0].key != null) output.attrs = {key: args[0].key}
		return output
	}
	m.component = function(component) {
		return parameterize(component, [].slice.call(arguments, 1))
	}
	m.mount = m.module = function(root, component) {
		if (!root) throw new Error("Please ensure the DOM element exists before rendering a template into it.");
		var index = roots.indexOf(root);
		if (index < 0) index = roots.length;
		
		var isPrevented = false;
		var event = {preventDefault: function() {
			isPrevented = true;
			computePreRedrawHook = computePostRedrawHook = null;
		}};
		for (var i = 0, unloader; unloader = unloaders[i]; i++) {
			unloader.handler.call(unloader.controller, event)
			unloader.controller.onunload = null
		}
		if (isPrevented) {
			for (var i = 0, unloader; unloader = unloaders[i]; i++) unloader.controller.onunload = unloader.handler
		}
		else unloaders = []
		
		if (controllers[index] && typeof controllers[index].onunload === FUNCTION) {
			controllers[index].onunload(event)
		}
		
		if (!isPrevented) {
			m.redraw.strategy("all");
			m.startComputation();
			roots[index] = root;
			if (arguments.length > 2) component = subcomponent(component, [].slice.call(arguments, 2))
			var currentComponent = topComponent = component = component || {controller: function() {}};
			var constructor = component.controller || noop
			var controller = new constructor;
			//controllers may call m.mount recursively (via m.route redirects, for example)
			//this conditional ensures only the last recursive m.mount call is applied
			if (currentComponent === topComponent) {
				controllers[index] = controller;
				components[index] = component
			}
			endFirstComputation();
			return controllers[index]
		}
	};
	var redrawing = false
	m.redraw = function(force) {
		if (redrawing) return
		redrawing = true
		//lastRedrawId is a positive number if a second redraw is requested before the next animation frame
		//lastRedrawID is null if it's the first redraw and not an event handler
		if (lastRedrawId && force !== true) {
			//when setTimeout: only reschedule redraw if time between now and previous redraw is bigger than a frame, otherwise keep currently scheduled timeout
			//when rAF: always reschedule redraw
			if ($requestAnimationFrame === window.requestAnimationFrame || new Date - lastRedrawCallTime > FRAME_BUDGET) {
				if (lastRedrawId > 0) $cancelAnimationFrame(lastRedrawId);
				lastRedrawId = $requestAnimationFrame(redraw, FRAME_BUDGET)
			}
		}
		else {
			redraw();
			lastRedrawId = $requestAnimationFrame(function() {lastRedrawId = null}, FRAME_BUDGET)
		}
		redrawing = false
	};
	m.redraw.strategy = m.prop();
	function redraw() {
		if (computePreRedrawHook) {
			computePreRedrawHook()
			computePreRedrawHook = null
		}
		for (var i = 0, root; root = roots[i]; i++) {
			if (controllers[i]) {
				var args = components[i].controller && components[i].controller.$$args ? [controllers[i]].concat(components[i].controller.$$args) : [controllers[i]]
				m.render(root, components[i].view ? components[i].view(controllers[i], args) : "")
			}
		}
		//after rendering within a routed context, we need to scroll back to the top, and fetch the document title for history.pushState
		if (computePostRedrawHook) {
			computePostRedrawHook();
			computePostRedrawHook = null
		}
		lastRedrawId = null;
		lastRedrawCallTime = new Date;
		m.redraw.strategy("diff")
	}

	var pendingRequests = 0;
	m.startComputation = function() {pendingRequests++};
	m.endComputation = function() {
		pendingRequests = Math.max(pendingRequests - 1, 0);
		if (pendingRequests === 0) m.redraw()
	};
	var endFirstComputation = function() {
		if (m.redraw.strategy() == "none") {
			pendingRequests--
			m.redraw.strategy("diff")
		}
		else m.endComputation();
	}

	m.withAttr = function(prop, withAttrCallback) {
		return function(e) {
			e = e || event;
			var currentTarget = e.currentTarget || this;
			withAttrCallback(prop in currentTarget ? currentTarget[prop] : currentTarget.getAttribute(prop))
		}
	};

	//routing
	var modes = {pathname: "", hash: "#", search: "?"};
	var redirect = noop, routeParams, currentRoute, isDefaultRoute = false;
	m.route = function() {
		//m.route()
		if (arguments.length === 0) return currentRoute;
		//m.route(el, defaultRoute, routes)
		else if (arguments.length === 3 && type.call(arguments[1]) === STRING) {
			var root = arguments[0], defaultRoute = arguments[1], router = arguments[2];
			redirect = function(source) {
				var path = currentRoute = normalizeRoute(source);
				if (!routeByValue(root, router, path)) {
					if (isDefaultRoute) throw new Error("Ensure the default route matches one of the routes defined in m.route")
					isDefaultRoute = true
					m.route(defaultRoute, true)
					isDefaultRoute = false
				}
			};
			var listener = m.route.mode === "hash" ? "onhashchange" : "onpopstate";
			window[listener] = function() {
				var path = $location[m.route.mode]
				if (m.route.mode === "pathname") path += $location.search
				if (currentRoute != normalizeRoute(path)) {
					redirect(path)
				}
			};
			computePreRedrawHook = setScroll;
			window[listener]()
		}
		//config: m.route
		else if (arguments[0].addEventListener || arguments[0].attachEvent) {
			var element = arguments[0];
			var isInitialized = arguments[1];
			var context = arguments[2];
			var vdom = arguments[3];
			element.href = (m.route.mode !== 'pathname' ? $location.pathname : '') + modes[m.route.mode] + vdom.attrs.href;
			if (element.addEventListener) {
				element.removeEventListener("click", routeUnobtrusive);
				element.addEventListener("click", routeUnobtrusive)
			}
			else {
				element.detachEvent("onclick", routeUnobtrusive);
				element.attachEvent("onclick", routeUnobtrusive)
			}
		}
		//m.route(route, params, shouldReplaceHistoryEntry)
		else if (type.call(arguments[0]) === STRING) {
			var oldRoute = currentRoute;
			currentRoute = arguments[0];
			var args = arguments[1] || {}
			var queryIndex = currentRoute.indexOf("?")
			var params = queryIndex > -1 ? parseQueryString(currentRoute.slice(queryIndex + 1)) : {}
			for (var i in args) params[i] = args[i]
			var querystring = buildQueryString(params)
			var currentPath = queryIndex > -1 ? currentRoute.slice(0, queryIndex) : currentRoute
			if (querystring) currentRoute = currentPath + (currentPath.indexOf("?") === -1 ? "?" : "&") + querystring;

			var shouldReplaceHistoryEntry = (arguments.length === 3 ? arguments[2] : arguments[1]) === true || oldRoute === arguments[0];

			if (window.history.pushState) {
				computePreRedrawHook = setScroll
				computePostRedrawHook = function() {
					window.history[shouldReplaceHistoryEntry ? "replaceState" : "pushState"](null, $document.title, modes[m.route.mode] + currentRoute);
				};
				redirect(modes[m.route.mode] + currentRoute)
			}
			else {
				$location[m.route.mode] = currentRoute
				redirect(modes[m.route.mode] + currentRoute)
			}
		}
	};
	m.route.param = function(key) {
		if (!routeParams) throw new Error("You must call m.route(element, defaultRoute, routes) before calling m.route.param()")
		return routeParams[key]
	};
	m.route.mode = "search";
	function normalizeRoute(route) {
		return route.slice(modes[m.route.mode].length)
	}
	function routeByValue(root, router, path) {
		routeParams = {};

		var queryStart = path.indexOf("?");
		if (queryStart !== -1) {
			routeParams = parseQueryString(path.substr(queryStart + 1, path.length));
			path = path.substr(0, queryStart)
		}

		// Get all routes and check if there's
		// an exact match for the current path
		var keys = Object.keys(router);
		var index = keys.indexOf(path);
		if(index !== -1){
			m.mount(root, router[keys [index]]);
			return true;
		}

		for (var route in router) {
			if (route === path) {
				m.mount(root, router[route]);
				return true
			}

			var matcher = new RegExp("^" + route.replace(/:[^\/]+?\.{3}/g, "(.*?)").replace(/:[^\/]+/g, "([^\\/]+)") + "\/?$");

			if (matcher.test(path)) {
				path.replace(matcher, function() {
					var keys = route.match(/:[^\/]+/g) || [];
					var values = [].slice.call(arguments, 1, -2);
					for (var i = 0, len = keys.length; i < len; i++) routeParams[keys[i].replace(/:|\./g, "")] = decodeURIComponent(values[i])
					m.mount(root, router[route])
				});
				return true
			}
		}
	}
	function routeUnobtrusive(e) {
		e = e || event;
		if (e.ctrlKey || e.metaKey || e.which === 2) return;
		if (e.preventDefault) e.preventDefault();
		else e.returnValue = false;
		var currentTarget = e.currentTarget || e.srcElement;
		var args = m.route.mode === "pathname" && currentTarget.search ? parseQueryString(currentTarget.search.slice(1)) : {};
		while (currentTarget && currentTarget.nodeName.toUpperCase() != "A") currentTarget = currentTarget.parentNode
		m.route(currentTarget[m.route.mode].slice(modes[m.route.mode].length), args)
	}
	function setScroll() {
		if (m.route.mode != "hash" && $location.hash) $location.hash = $location.hash;
		else window.scrollTo(0, 0)
	}
	function buildQueryString(object, prefix) {
		var duplicates = {}
		var str = []
		for (var prop in object) {
			var key = prefix ? prefix + "[" + prop + "]" : prop
			var value = object[prop]
			var valueType = type.call(value)
			var pair = (value === null) ? encodeURIComponent(key) :
				valueType === OBJECT ? buildQueryString(value, key) :
				valueType === ARRAY ? value.reduce(function(memo, item) {
					if (!duplicates[key]) duplicates[key] = {}
					if (!duplicates[key][item]) {
						duplicates[key][item] = true
						return memo.concat(encodeURIComponent(key) + "=" + encodeURIComponent(item))
					}
					return memo
				}, []).join("&") :
				encodeURIComponent(key) + "=" + encodeURIComponent(value)
			if (value !== undefined) str.push(pair)
		}
		return str.join("&")
	}
	function parseQueryString(str) {
		if (str.charAt(0) === "?") str = str.substring(1);
		
		var pairs = str.split("&"), params = {};
		for (var i = 0, len = pairs.length; i < len; i++) {
			var pair = pairs[i].split("=");
			var key = decodeURIComponent(pair[0])
			var value = pair.length == 2 ? decodeURIComponent(pair[1]) : null
			if (params[key] != null) {
				if (type.call(params[key]) !== ARRAY) params[key] = [params[key]]
				params[key].push(value)
			}
			else params[key] = value
		}
		return params
	}
	m.route.buildQueryString = buildQueryString
	m.route.parseQueryString = parseQueryString
	
	function reset(root) {
		var cacheKey = getCellCacheKey(root);
		clear(root.childNodes, cellCache[cacheKey]);
		cellCache[cacheKey] = undefined
	}

	m.deferred = function () {
		var deferred = new Deferred();
		deferred.promise = propify(deferred.promise);
		return deferred
	};
	function propify(promise, initialValue) {
		var prop = m.prop(initialValue);
		promise.then(prop);
		prop.then = function(resolve, reject) {
			return propify(promise.then(resolve, reject), initialValue)
		};
		return prop
	}
	//Promiz.mithril.js | Zolmeister | MIT
	//a modified version of Promiz.js, which does not conform to Promises/A+ for two reasons:
	//1) `then` callbacks are called synchronously (because setTimeout is too slow, and the setImmediate polyfill is too big
	//2) throwing subclasses of Error cause the error to be bubbled up instead of triggering rejection (because the spec does not account for the important use case of default browser error handling, i.e. message w/ line number)
	function Deferred(successCallback, failureCallback) {
		var RESOLVING = 1, REJECTING = 2, RESOLVED = 3, REJECTED = 4;
		var self = this, state = 0, promiseValue = 0, next = [];

		self["promise"] = {};

		self["resolve"] = function(value) {
			if (!state) {
				promiseValue = value;
				state = RESOLVING;

				fire()
			}
			return this
		};

		self["reject"] = function(value) {
			if (!state) {
				promiseValue = value;
				state = REJECTING;

				fire()
			}
			return this
		};

		self.promise["then"] = function(successCallback, failureCallback) {
			var deferred = new Deferred(successCallback, failureCallback);
			if (state === RESOLVED) {
				deferred.resolve(promiseValue)
			}
			else if (state === REJECTED) {
				deferred.reject(promiseValue)
			}
			else {
				next.push(deferred)
			}
			return deferred.promise
		};

		function finish(type) {
			state = type || REJECTED;
			next.map(function(deferred) {
				state === RESOLVED && deferred.resolve(promiseValue) || deferred.reject(promiseValue)
			})
		}

		function thennable(then, successCallback, failureCallback, notThennableCallback) {
			if (((promiseValue != null && type.call(promiseValue) === OBJECT) || typeof promiseValue === FUNCTION) && typeof then === FUNCTION) {
				try {
					// count protects against abuse calls from spec checker
					var count = 0;
					then.call(promiseValue, function(value) {
						if (count++) return;
						promiseValue = value;
						successCallback()
					}, function (value) {
						if (count++) return;
						promiseValue = value;
						failureCallback()
					})
				}
				catch (e) {
					m.deferred.onerror(e);
					promiseValue = e;
					failureCallback()
				}
			} else {
				notThennableCallback()
			}
		}

		function fire() {
			// check if it's a thenable
			var then;
			try {
				then = promiseValue && promiseValue.then
			}
			catch (e) {
				m.deferred.onerror(e);
				promiseValue = e;
				state = REJECTING;
				return fire()
			}
			thennable(then, function() {
				state = RESOLVING;
				fire()
			}, function() {
				state = REJECTING;
				fire()
			}, function() {
				try {
					if (state === RESOLVING && typeof successCallback === FUNCTION) {
						promiseValue = successCallback(promiseValue)
					}
					else if (state === REJECTING && typeof failureCallback === "function") {
						promiseValue = failureCallback(promiseValue);
						state = RESOLVING
					}
				}
				catch (e) {
					m.deferred.onerror(e);
					promiseValue = e;
					return finish()
				}

				if (promiseValue === self) {
					promiseValue = TypeError();
					finish()
				}
				else {
					thennable(then, function () {
						finish(RESOLVED)
					}, finish, function () {
						finish(state === RESOLVING && RESOLVED)
					})
				}
			})
		}
	}
	m.deferred.onerror = function(e) {
		if (type.call(e) === "[object Error]" && !e.constructor.toString().match(/ Error/)) throw e
	};

	m.sync = function(args) {
		var method = "resolve";
		function synchronizer(pos, resolved) {
			return function(value) {
				results[pos] = value;
				if (!resolved) method = "reject";
				if (--outstanding === 0) {
					deferred.promise(results);
					deferred[method](results)
				}
				return value
			}
		}

		var deferred = m.deferred();
		var outstanding = args.length;
		var results = new Array(outstanding);
		if (args.length > 0) {
			for (var i = 0; i < args.length; i++) {
				args[i].then(synchronizer(i, true), synchronizer(i, false))
			}
		}
		else deferred.resolve([]);

		return deferred.promise
	};
	function identity(value) {return value}

	function ajax(options) {
		if (options.dataType && options.dataType.toLowerCase() === "jsonp") {
			var callbackKey = "mithril_callback_" + new Date().getTime() + "_" + (Math.round(Math.random() * 1e16)).toString(36);
			var script = $document.createElement("script");

			window[callbackKey] = function(resp) {
				script.parentNode.removeChild(script);
				options.onload({
					type: "load",
					target: {
						responseText: resp
					}
				});
				window[callbackKey] = undefined
			};

			script.onerror = function(e) {
				script.parentNode.removeChild(script);

				options.onerror({
					type: "error",
					target: {
						status: 500,
						responseText: JSON.stringify({error: "Error making jsonp request"})
					}
				});
				window[callbackKey] = undefined;

				return false
			};

			script.onload = function(e) {
				return false
			};

			script.src = options.url
				+ (options.url.indexOf("?") > 0 ? "&" : "?")
				+ (options.callbackKey ? options.callbackKey : "callback")
				+ "=" + callbackKey
				+ "&" + buildQueryString(options.data || {});
			$document.body.appendChild(script)
		}
		else {
			var xhr = new window.XMLHttpRequest;
			xhr.open(options.method, options.url, true, options.user, options.password);
			xhr.onreadystatechange = function() {
				if (xhr.readyState === 4) {
					if (xhr.status >= 200 && xhr.status < 300) options.onload({type: "load", target: xhr});
					else options.onerror({type: "error", target: xhr})
				}
			};
			if (options.serialize === JSON.stringify && options.data && options.method !== "GET") {
				xhr.setRequestHeader("Content-Type", "application/json; charset=utf-8")
			}
			if (options.deserialize === JSON.parse) {
				xhr.setRequestHeader("Accept", "application/json, text/*");
			}
			if (typeof options.config === FUNCTION) {
				var maybeXhr = options.config(xhr, options);
				if (maybeXhr != null) xhr = maybeXhr
			}

			var data = options.method === "GET" || !options.data ? "" : options.data
			if (data && (type.call(data) != STRING && data.constructor != window.FormData)) {
				throw "Request data should be either be a string or FormData. Check the `serialize` option in `m.request`";
			}
			xhr.send(data);
			return xhr
		}
	}
	function bindData(xhrOptions, data, serialize) {
		if (xhrOptions.method === "GET" && xhrOptions.dataType != "jsonp") {
			var prefix = xhrOptions.url.indexOf("?") < 0 ? "?" : "&";
			var querystring = buildQueryString(data);
			xhrOptions.url = xhrOptions.url + (querystring ? prefix + querystring : "")
		}
		else xhrOptions.data = serialize(data);
		return xhrOptions
	}
	function parameterizeUrl(url, data) {
		var tokens = url.match(/:[a-z]\w+/gi);
		if (tokens && data) {
			for (var i = 0; i < tokens.length; i++) {
				var key = tokens[i].slice(1);
				url = url.replace(tokens[i], data[key]);
				delete data[key]
			}
		}
		return url
	}

	m.request = function(xhrOptions) {
		if (xhrOptions.background !== true) m.startComputation();
		var deferred = new Deferred();
		var isJSONP = xhrOptions.dataType && xhrOptions.dataType.toLowerCase() === "jsonp";
		var serialize = xhrOptions.serialize = isJSONP ? identity : xhrOptions.serialize || JSON.stringify;
		var deserialize = xhrOptions.deserialize = isJSONP ? identity : xhrOptions.deserialize || JSON.parse;
		var extract = isJSONP ? function(jsonp) {return jsonp.responseText} : xhrOptions.extract || function(xhr) {
			return xhr.responseText.length === 0 && deserialize === JSON.parse ? null : xhr.responseText
		};
		xhrOptions.method = (xhrOptions.method || 'GET').toUpperCase();
		xhrOptions.url = parameterizeUrl(xhrOptions.url, xhrOptions.data);
		xhrOptions = bindData(xhrOptions, xhrOptions.data, serialize);
		xhrOptions.onload = xhrOptions.onerror = function(e) {
			try {
				e = e || event;
				var unwrap = (e.type === "load" ? xhrOptions.unwrapSuccess : xhrOptions.unwrapError) || identity;
				var response = unwrap(deserialize(extract(e.target, xhrOptions)), e.target);
				if (e.type === "load") {
					if (type.call(response) === ARRAY && xhrOptions.type) {
						for (var i = 0; i < response.length; i++) response[i] = new xhrOptions.type(response[i])
					}
					else if (xhrOptions.type) response = new xhrOptions.type(response)
				}
				deferred[e.type === "load" ? "resolve" : "reject"](response)
			}
			catch (e) {
				m.deferred.onerror(e);
				deferred.reject(e)
			}
			if (xhrOptions.background !== true) m.endComputation()
		};
		ajax(xhrOptions);
		deferred.promise = propify(deferred.promise, xhrOptions.initialValue);
		return deferred.promise
	};

	//testing API
	m.deps = function(mock) {
		initialize(window = mock || window);
		return window;
	};
	//for internal testing only, do not use `m.deps.factory`
	m.deps.factory = app;

	return m
})(typeof window != "undefined" ? window : {});

if (typeof module != "undefined" && module !== null && module.exports) module.exports = m;
else if (typeof define === "function" && define.amd) define(function() {return m});
var Phone = function() {
	var _this = this;
	_this.number = m.prop("");
	_this.format = function() {
		var a1 = _this.number().slice(0, 2);
		var a2 = " (" + _this.number().slice(2, 5) + ") ";
		var a3 = _this.number().slice(5, 8) + "-"
		return a1 + a2 + a3 + _this.number().slice(8)
	};
	return _this;
};
Phone.listView = function(list) {
	return m("table", {
		class: "table"
	}, [
		m("thead", [
			m("tr", [
				m("th", "Number")
			])
		]),
		m("tbody", [
			list.data().map(function(item) {
				var phone = new Phone();
				phone.number(item.Number);
				return m("tr", {
					"attr-id": item.Id
				}, [
					m("td", phone.format()),
					m("td", {
						class: "text-right"
					}, [
						m("img", {
							class: "icon icon-xs icon-delete",
							src: "/public/images/icon_delete.svg",
							onclick: function() {
								var c = confirm("Delete this number?");
								if (c) {
									// TODO delete from database
									console.log("not implemented");
								}
							}
						})
					])
				])
			})
		])
	]);
};
var Profile = {
	signout: function(ev) {
		ev.preventDefault();
		cookie.removeItem("id");
		cookie.removeItem("session_token");
		m.route("/login");
	},
	data: function(uid) {
		return m.request({
			method: "GET",
			url: "/api/profile.json?uid=" + uid
		});
	},
	sendView: function(uid) {
		return m.request({
			method: "PUT",
			url: "/api/profile.json",
			data: { UserID: parseInt(uid, 10) }
		});
	}
};

Profile.controller = function() {
	if (cookie.getItem("id") === null) {
		return m.route("/login");
	}
	var _this = this;
	_this.username = m.prop("");
	_this.email = m.prop("");
	_this.phones = new List({type: Phone});
	_this.cards = new List({type: Card});
	var userId = cookie.getItem("id");
	Profile.data(userId).then(function(data) {
		_this.email(data.Email);
		_this.username(data.Name);
		_this.phones.userId(userId);
		_this.phones.data(data.Phones);
		_this.cards.userId(userId);
		_this.cards.data(data.Cards);
	}, function(err) {
		console.error(err);
	});
	Profile.sendView(userId);
};

Profile.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Profile.viewFull(controller),
		Footer.view()
	]);
};

Profile.viewFull = function(controller) {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-12"
			}, [
				m("h1", "Profile")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-md-7 margin-top-sm"
			}, [
				m("h3", "Account Details"),
				m("form", {
					class: "margin-top-sm"
				}, [
					m("div", {
						class: "card"
					}, [
						m("div", {
							class: "form-group"
						}, [
							m("label", "Username"),
							m("div", [
								m("div", controller.email())
							])
						]),
						m("div", {
							class: "form-group"
						}, [
							m("label", "Password"),
							m("div", [
								m("a", {
									href: "#"
								}, "Change password")
							])
						]),
						m("div", {
							class: "form-group"
						}, [
							m("label", {
								for: "username"
							}, "Name"),
							m("div", [
								m("input", {
									id: "username",
									type: "text",
									class: "form-control",
									value: controller.username()
								})
							])
						]),
						m("div", {
							class: "form-group margin-top-sm"
						}, [
							m("div", [
								m("a", {
									class: "btn btn-sm",
									href: "#/",
									onclick: Profile.signout
								}, "Sign out")
							])
						])
					]),
					m("h3", {
						class: "margin-top-sm"
					}, "Phone numbers"),
					m("div", {
						class: "form-group card"
					}, [
						m("div", [
							controller.phones.view()
						])
					]),
					m("h3", {
						class: "margin-top-sm"
					}, "Credit cards"),
					m("div", {
						class: "form-group card"
					}, [
						m("div", [
							controller.cards.view(),
							m("div", [
								m("a", {
									id: controller.cards.id + "-add-btn",
									class: "btn btn-sm",
									href: "/cards/new",
									config: m.route
								}, "+Add Card")
							])
						])
					])
				])
			])
		])
	]);
};
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
var Signup = {
	signup: function(ev) {
		ev.preventDefault();
		var name = document.getElementById("name").value;
		var email = document.getElementById("email").value;
		var pass = document.getElementById("password").value;
		var flexId = document.getElementById("phone").value;
		return m.request({
			method: "POST",
			data: {
				name: name,
				email: email,
				password: pass,
				fid: flexId
			},
			url: "/api/signup.json"
		}).then(function(data) {
			var date = new Date();
			var exp = date.setDate(date + 30);
			cookie.setItem("id", data.Id, exp, null, null, false);
			cookie.setItem("session_token", data.SessionToken, exp, null, null, false);
			m.route("/profile");
		}, function(err) {
			Signup.controller.error(err.Msg);
		});
	}
};

Signup.controller = function() {
	Login.checkAuth(function(cb) {
		if (cb) {
			return m.route("/profile");
		}
	});
	var name = m.route.param("name") || "";
	var phone = m.route.param("fid") || "";
	Signup.controller.userName = m.prop(name);
	Signup.controller.phone = m.prop(phone);
	Signup.controller.error = m.prop("");
};

Signup.vm = {
	phoneDisabled: function() {
		return Signup.controller.phone().length > 0;
	}
};

Signup.view = function() {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Signup.viewFull(),
		Footer.view()
	]);
};

Signup.viewFull = function() {
	return m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-md-push-3 col-md-6 card"
			}, [
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("h2", "Sign Up")
					])
				]),
				m("form", {
					onsubmit: Signup.signup
				}, [
					m("div", {
						class: "row margin-top-sm"
					}, [
						m("div", {
							class: "col-md-12"
						}, [

							function() {
								if (Signup.controller.error() !== "") {
									return m("div", {
										class: "alert alert-danger"
									}, Signup.controller.error());
								}
							}(),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "text",
									class: "form-control",
									id: "name",
									placeholder: "Your name"
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "tel",
									class: "form-control",
									id: "phone",
									placeholder: "Your phone number",
									value: Signup.controller.phone(),
									disabled: Signup.vm.phoneDisabled()
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "email",
									class: "form-control",
									id: "email",
									placeholder: "Email"
								})
							]),
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									type: "password",
									class: "form-control",
									id: "password",
									placeholder: "Password"
								})
							])
						])
					]),
					m("div", {
						class: "row"
					}, [
						m("div", {
							class: "col-md-12 text-center"
						}, [
							m("div", {
								class: "form-group"
							}, [
								m("input", {
									class: "btn btn-sm",
									id: "btn",
									type: "submit",
									value: "Sign Up"
								})
							])
						])
					])
				]),
				m("div", {
					class: "row"
				}, [
					m("div", {
						class: "col-md-12 text-center"
					}, [
						m("div", {
							class: "form-group"
						}, [
							m("span", "Have an account? "),
							m("a", {
								href: "/login",
								config: m.route
							}, "Log In")
						])
					])
				])
			])
		])
	]);
};
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
var Trainer = function() {
	Trainer.id = m.prop(0);
	Trainer.sentence = function(id) {
		var url = "/api/sentence.json";
		if (id !== undefined) {
			url += "?id=" + id;
		}
		return m.request({
			method: "GET",
			url: url
		})
	};
	Trainer.save = function() {
		var sentence = "";
		for (var i = 0; i < Train.vm.words.length; ++i) {
			var word = Train.vm.words[i];
			sentence += word.type() + " ";
		}
		var data = {
			ID: Train.vm.state.id,
			AssignmentID: Train.vm.state.assignmentId,
			ForeignID: Train.vm.state.foreignId,
			Sentence: sentence,
			MaxAssignments: Train.vm.state.maxAssignments
		};
		return m.request({
			method: "PUT",
			url: "/api/sentence.json",
			data: data
		});
	};
};

var Word = function(word) {
	var _this = this;
	Word.value = m.prop(word);
	Word.type = m.prop("_N(" + _this.value() + ")");
	Word.setClass = function() {
		if (Word.classList.length > 0) {
			_this.type("_N(" + _this.value() + ")");
			return Word.className = "";
		}
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				_this.type("_C(" + _this.value() + ")");
				return Word.classList.add("red");
			case "OBJECTS":
				_this.type("_O(" + _this.value() + ")");
				return Word.classList.add("blue");
			case "ACTORS":
				_this.type("_A(" + _this.value() + ")");
				return Word.classList.add("green");
			case "TIMES":
				_this.type("_T(" + _this.value() + ")");
				return Word.classList.add("yellow");
			case "PLACES":
				_this.type("_P(" + _this.value() + ")");
				return Word.classList.add("pink");
			default:
				console.error(
					"invalid training state: " +
					Train.vm.trainingCategory()
				);
		};
	};
};

var Train = {};
Train.controller = function() {
	var id = m.route.param("sentenceID");
	Train.controller.trainer = new Trainer();
	Train.controller.trainer.sentence(id).then(function(data) {
		Train.vm.init(data);
	});
};

Train.vm = {
	init: function(data) {
		Train.vm.state = {};
		Train.vm.trainer = new Trainer();
		Train.vm.trainingCategory = m.prop("COMMANDS");
		Train.vm.state.id = data.ID;
		Train.vm.state.assignmentId = m.route.param("assignmentId");
		Train.vm.state.foreignId = data.ForeignID;
		Train.vm.state.maxAssignments = data.MaxAssignments;
		Train.vm.words = [];
		var words = data.Sentence.split(/\s+/);
		for (var i = 0; i < words.length; ++i) {
			Train.vm.words[i] = new Word(words[i]);
		}
		window.addEventListener("keypress", function(ev) {
			if (ev.keyCode === 102 /* 'f' key */ ) {
				ev.preventDefault();
				Train.vm.nextCategory();
			} else if (ev.keyCode === 98 /* 'b' key */ ) {
				ev.preventDefault();
				Train.vm.prevCategory();
			}
		});
	},
	nextCategory: function() {
		var el = document.getElementById("training-category");
		var helpTitle = document.getElementById("help-title");
		var helpBody = document.getElementById("help-body");
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				Train.vm.trainingCategory("OBJECTS");
				var btn = document.getElementById("back-btn");
				btn.classList.remove("hidden");
				helpTitle.innerText = "What is an object? ";
				helpBody.innerText = "Objects are the direct objects of the sentence.";
				break;
			case "OBJECTS":
				Train.vm.trainingCategory("ACTORS");
				helpTitle.innerText = "What is an actor? ";
				helpBody.innerText = "Actors are often the indirect objects of the sentence.";
				break;
			case "ACTORS":
				Train.vm.trainingCategory("TIMES");
				helpTitle.innerText = "What are times? ";
				helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. This Wednesday. Etc.";
				break;
			case "TIMES":
				Train.vm.trainingCategory("PLACES");
				var btn = document.getElementById("continue-btn");
				helpTitle.innerText = "What are places? ";
				helpBody.innerText = "A place is any description of where an event should take place. Starbucks. Nearby. Etc.";
				btn.innerText = "Save";
				break;
			case "PLACES":
				var btn = document.getElementById("continue-btn");
				if (btn.innerText !== "Saving..." && btn.innerText !== "Thank you!") {
					Train.vm.save();
					Train.controller.trainer.save().then(function() {
						Train.vm.saveComplete();
						setTimeout(function() {
							m.route("/train");
						}, 2000);
					});
				}
				return;
		};
		el.innerText = Train.vm.trainingCategory();
		el.className = Train.vm.categoryColor();
	},
	prevCategory: function() {
		var el = document.getElementById("training-category");
		var helpTitle = document.getElementById("help-title");
		var helpBody = document.getElementById("help-body");
		switch (Train.vm.trainingCategory()) {
			case "OBJECTS":
				Train.vm.trainingCategory("COMMANDS");
				helpTitle.innerText = "What is a command? ";
				helpBody.innerText = 'A command is a verb, like "Find," "Walk," or "Meet."';
				var btn = document.getElementById("back-btn");
				btn.classList.add("hidden");
				break;
			case "ACTORS":
				Train.vm.trainingCategory("OBJECTS");
				helpTitle.innerText = "What is an object? ";
				helpBody.innerText = "Objects are the direct objects of the sentence.";
				break;
			case "TIMES":
				Train.vm.trainingCategory("ACTORS");
				helpTitle.innerText = "What is an actor? ";
				helpBody.innerText = "Actors are often the indirect objects of the sentence.";
				break;
			case "PLACES":
				Train.vm.trainingCategory("TIMES");
				helpTitle.innerText = "What are times? ";
				helpBody.innerText = "Every Tuesday. Noon. Friday. Tomorrow. This Wednesday. Etc.";
				var btn = document.getElementById("continue-btn");
				btn.innerText = "Continue";
				break;
		};
		el.innerText = Train.vm.trainingCategory();
		el.className = Train.vm.categoryColor();
	},
	categoryColor: function() {
		switch (Train.vm.trainingCategory()) {
			case "COMMANDS":
				return "red";
			case "OBJECTS":
				return "blue";
			case "ACTORS":
				return "green";
			case "TIMES":
				return "yellow";
			case "PLACES":
				return "pink";
		};
	},
	save: function() {
		var btn = document.getElementById("continue-btn");
		btn.innerText = "Saving...";
		btn = document.getElementById("back-btn");
		btn.classList.add("hidden");
	},
	saveComplete: function() {
		var btn = document.getElementById("continue-btn");
		btn.innerText = "Thank you!";
	}
};

Train.view = function(controller) {
	return m("div", {
		class: "body"
	}, [
		header.view(),
		Train.viewFull(),
		Train.viewEmpty(),
		Footer.view()
	]);
};

Train.viewFull = function() {
	var view = m("div", {
		id: "full",
		class: "container"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-sm-12"
			}, [
				m("h1", "Train"),
				m("p", "Train Ava to understand language.")
			])
		]),
		m("div", {
			class: "row"
		}, [
			m("div", {
				class: "col-sm-12 text-right"
			}, [
				m("a", {
					class: "btn",
					onclick: function() {
						m.route("/train");
					}
				}, "Skip Sentence"),
			]),
			m("div", {
				class: "col-sm-12 margin-top-sm"
			}, [
				m("h2",
					"Tap the ",
					m("span", {
						id: "training-category",
						class: Train.vm.categoryColor()
					}, Train.vm.trainingCategory()),
					" in this sentence:"
				),
				m("p", {
					class: "light"
				}, [
					m("strong",
						m("i", {
							id: "help-title"
						}, "What is a command? ")
					),
					m("span", {
						id: "help-body"
					}, 'A command is a verb, like "Find," "Walk," or "Meet."')
				])
			]),
			m("div", {
				class: "col-sm-12"
			}, [
				m("p", {
					id: "train-sentence",
					class: "big no-select"
				}, [
					Train.vm.words.map(function(word, i) {
						return [
							m("span", {
								onclick: word.setClass,
							}, word.value()),
							m("span", " ")
						]
					})
				])
			]),
			m("div", {
				class: "col-sm-12 text-right"
			}, [
				m("a", {
					id: "back-btn",
					href: "#/",
					class: "btn hidden",
					onclick: Train.vm.prevCategory
				}, "Go back"),
				m("a", {
					id: "continue-btn",
					href: "#/",
					class: "btn btn-primary btn-lg",
					onclick: Train.vm.nextCategory
				}, "Continue")
			])
		])
	]);
	if (Train.vm.state.id > 0) {
		return view;
	}
};

Train.viewEmpty = function() {
	var view = m("div", {
		id: "empty",
		class: "container jumbo"
	}, [
		m("div", {
			class: "row margin-top-sm"
		}, [
			m("div", {
				class: "col-sm-12 text-center"
			}, [
				m("h2", "All done!"),
				m("p", {
					style: "color:black"
				}, "No tasks need to be completed.")
			])
		])
	]);
	if (Train.vm.state.id === 0 || Train.vm.state.id === undefined) {
		return view;
	}
};
