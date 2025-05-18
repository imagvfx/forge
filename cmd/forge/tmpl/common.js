function printStatus(s) {
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.remove("error");
	statusBar.innerHTML = s;
}

function printErrorStatus(e) {
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.add("error");
	statusBar.innerHTML = e;
}

function clearStatus() {
	printStatus("");
}

function submitAPI(form) {
	let api = form.action;
	let data = new FormData(form)
	postForge(api, data, function(_, err) {
		if (err) {
			printErrorStatus(err);
			return;
		}
		location.reload();
	})
	return false;
}

function postForge(api, data, handler) {
	let r = new XMLHttpRequest();
	r.open("post", api);
	r.send(data);
	r.onerror = function() {
		handler(null, "network error occurred. please check whether the server is down.");
	}
	r.onload = function() {
		let j = JSON.parse(r.responseText);
		if (j.Err != "") {
			handler(null, j.Err);
			return;
		}
		// j.Msg will be null, if it was an update operation.
		handler(j.Msg, null);
	}
}
