function getJsonFromUrl(hashBased) {
	var query;
	if (hashBased) {
		var pos = location.href.indexOf("?");
		if (pos==-1) return [];
		query = location.href.substr(pos+1);
	} else {
		query = location.search.substr(1);
	}
	var result = {};
	query.split("&").forEach(function(part) {
		if (!part) return;
		var item = part.split("=");
		var key = item[0];
		var val = decodeURIComponent(item[1]);
		var from = key.indexOf("[");
		if (from==-1) {
			result[key] = val;
		} else {
			var to = key.indexOf("]");
			var index = key.substring(from+1,to);
			key = key.substring(0,from);
			if (!result[key]) {
				result[key] = [];
			}
			if (!index) {
				result[key].push(val);
			} else {
				result[key][index] = val;
			}
		}
	});
	return result;
}

window.onload = function() {
	var params = getJsonFromUrl(window.location.search);
	if (document.querySelector("#signup") !== null) {
		document.querySelector("#flexid").value = params.flexid;
		document.querySelector("#flexidtype").value = params.flexidtype;
		var link = document.querySelector("#login");
		link.setAttribute("href", link.attributes.href+window.location.search);
	} else if (document.querySelector("#login") !== null) {
		var link = document.querySelector("#signup");
		link.setAttribute("href", link.attributes.href+window.location.search);
	}
};
