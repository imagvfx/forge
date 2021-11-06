"use strict";

window.onload = function() {
	let currentContextMenuLoader = null;
	document.onclick = function(event) {
		let selectStatusMenus = document.getElementsByClassName("selectStatusMenu");
		for (let menu of selectStatusMenus) {
			menu.style.visibility = "hidden";
		}
		let userMenu = document.getElementById("userAutoCompleteMenu");
		userMenu.replaceChildren();
		userMenu.style.visibility = "hidden";
		let infoMenu = document.getElementById("infoContextMenu");
		if (currentContextMenuLoader != null) {
			infoMenu.style.visibility = "hidden"
			currentContextMenuLoader = null;
			return;
		}
	}
	let allInputs = document.getElementsByTagName("input");
	for (let input of allInputs) {
		input.autocomplete = "off";
	}
	let inputs = document.getElementsByClassName("valueInput");
	for (let input of inputs) {
		input.onkeydown = function(ev) {
			if ((ev.ctrlKey && ev.code == "Enter") || ev.code == "NumpadEnter") {
				submitUpdaterOrAdder(ev, input);
			}
		}
		input.parentElement.onsubmit = function(ev) {
			submitUpdaterOrAdder(ev, input);
		}
	}
	for (let input of inputs) {
		input.oninput = function() {
			resizeTextArea(input);
		}
	}
	let uploadExcelInput = document.getElementById("uploadExcelInput");
	if (uploadExcelInput != null) {
		uploadExcelInput.onchange = function() {
			let uploadExcelForm = document.getElementById("uploadExcelForm");
			uploadExcelForm.submit();
		}
	}
	let pinnedPaths = document.getElementsByClassName("pinnedPathLink");
	for (let pp of pinnedPaths) {
		pp.onclick = function(event) {
			window.location.href = pp.dataset.link;
		}
		pp.ondragstart = function(event) {
			event.dataTransfer.effectAllowed = "move";
			let zone = document.getElementById("pinnedPathDropZone");
			// TODO: extract common code to drag-drop for zones as functions.
			zone.ondragover = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "move";
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let p of paths) {
					if (p == pp) {
						continue;
					}
					if (event.clientY < p.offsetTop + p.offsetHeight/2) {
						p.parentNode.insertBefore(pp, p);
						break;
					}
					if (p.nextElementSibling == null) {
						p.parentNode.appendChild(pp, p);
						break;
					}
				}
			}
			zone.ondragleave = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "none";
			}
			zone.ondrop = function(event) {
				let at = -1;
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let i = 0; i < paths.length; i++) {
					let p = paths[i];
					if (p == pp) {
						at = i;
						break;
					}
				}
				if (at == -1) {
					console.log("unable to find drop target from pinned paths");
					return;
				}
				updatePinnedPath(pp.innerText, at);
			}
			let del = document.getElementById("pinnedPathDeleteButton");
			del.style.display = "inline-block";
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.style.color = "#A22";
				del.style.border = "1px solid #A22";
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.style.color = "#DDD";
				del.style.border = "1px solid #DDD";
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				updatePinnedPath(pp.innerText, -1);
			}
		}
		pp.ondragend = function(event) {
			let zone = document.getElementById("pinnedPathDropZone");
			removeDragDropEvents(zone);
			let del = document.getElementById("pinnedPathDeleteButton");
			removeDragDropEvents(del);
			del.style.display = "none";
			if (event.dataTransfer.dropEffect == "none") {
				let curIdx = -1;
				let paths = document.getElementsByClassName("pinnedPathLink");
				for (let i = 0; i < paths.length; i++) {
					let p = paths[i];
					if (p == pp) {
						curIdx = i;
						break;
					}
				}
				if (curIdx == -1) {
					console.log("unable to find drop target from pinnedpaths");
					return;
				}
				let origIdx = parseInt(pp.dataset.idx);
				if (curIdx == origIdx) {
					return;
				}
				if (curIdx > origIdx) {
					pp.parentNode.insertBefore(pp, pp.parentNode.children[origIdx]);
					return;
				}
				if (curIdx < origIdx) {
					if (origIdx+1 == paths.length) {
						pp.parentNode.appendChild(pp);
						return;
					}
					pp.parentNode.insertBefore(pp, pp.parentNode.children[origIdx+1]);
					return;
				}
			}
		}
	}
	let quickSearches = document.getElementsByClassName("quickSearchLink");
	for (let qs of quickSearches) {
		qs.onclick = function(event) {
			// I've had hard time when I drag quickSearchLink while it is 'a' tag.
			// At first glance qs.ondragstart seemed to work consitently, then the link is clicked instead.
			// Hope I got peace by making quickSearchLink 'div'.
			window.location.href = qs.dataset.link;
			return;
		}
		qs.ondragstart = function(event) {
			event.dataTransfer.effectAllowed = "move";
			let zone = document.getElementById("quickSearchDropZone");
			zone.ondragover = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "move";
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let s of searches) {
					if (s == qs) {
						continue;
					}
					if (event.clientY < s.offsetTop || s.offsetTop + s.offsetHeight <= event.clientY) {
						continue;
					}
					if (event.clientX < s.offsetLeft + s.offsetWidth) {
						if (event.clientX < s.offsetLeft + s.offsetWidth/2) {
							s.parentNode.insertBefore(qs, s);
							break;
						}
						let next = s.nextElementSibling;
						if (next == null) {
							s.parentNode.appendChild(qs, s);
							break;
						}
						s.parentNode.insertBefore(qs, next);
						break;
					}
				}
			}
			zone.ondragleave = function(event) {
				event.preventDefault();
				event.dataTransfer.dropEffect = "none";
			}
			zone.ondrop = function(event) {
				let at = -1;
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let i = 0; i < searches.length; i++) {
					let s = searches[i];
					if (s == qs) {
						at = i;
						break;
					}
				}
				if (at == -1) {
					console.log("unable to find drop target from quicksearches");
					return;
				}
				updateQuickSearch(qs.innerText, at);
			}
			let del = document.getElementById("quickSearchDeleteButton");
			del.style.display = "inline-block";
			del.ondragenter = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "move";
				del.style.color = "#A22";
				del.style.border = "1px solid #A22";
			}
			del.ondragover = function(ev) {
				ev.preventDefault();
			}
			del.ondragleave = function(ev) {
				ev.preventDefault();
				ev.dataTransfer.dropEffect = "none";
				del.style.color = "#DDD";
				del.style.border = "1px solid #DDD";
			}
			del.ondrop = function(ev) {
				ev.preventDefault();
				ev.stopPropagation();
				updateQuickSearch(qs.innerText, -1);
			}
		}
		qs.ondragend = function(event) {
			let zone = document.getElementById("quickSearchDropZone");
			removeDragDropEvents(zone);
			let del = document.getElementById("quickSearchDeleteButton");
			removeDragDropEvents(del);
			del.style.display = "none";
			if (event.dataTransfer.dropEffect == "none") {
				let curIdx = -1;
				let searches = document.getElementsByClassName("quickSearchLink");
				for (let i = 0; i < searches.length; i++) {
					let s = searches[i];
					if (s == qs) {
						curIdx = i;
						break;
					}
				}
				if (curIdx == -1) {
					console.log("unable to find drop target from quicksearches");
					return;
				}
				let origIdx = parseInt(qs.dataset.idx);
				if (curIdx == origIdx) {
					return;
				}
				if (curIdx > origIdx) {
					qs.parentNode.insertBefore(qs, qs.parentNode.children[origIdx]);
					return;
				}
				if (curIdx < origIdx) {
					if (origIdx+1 == searches.length) {
						qs.parentNode.appendChild(qs);
						return;
					}
					qs.parentNode.insertBefore(qs, qs.parentNode.children[origIdx+1]);
					return;
				}
			}
		}
	}
	let currentStatusSelect = null;
	let statusSelects = document.getElementsByClassName("statusSelect");
	for (let sel of statusSelects) {
		let entType = sel.dataset.entryType;
		let menu = document.getElementById("selectStatusMenu-" + entType);
		if (menu == null) {
			// It can be null, if possible_status global for the entry type is not exists.
			continue
		}
		sel.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			if (currentStatusSelect == sel && menu.style.visibility == "visible") {
				menu.style.visibility = "hidden";
				currentStatusSelect = null;
				return;
			}
			currentStatusSelect = sel;
			// slight adjust of the menu position to make statusDots aligned.
			menu.style.visibility = "visible";
			menu.style.left = String(sel.offsetLeft - 6) + "px";
			menu.style.top = String(sel.offsetTop + sel.offsetHeight + 4) + "px";
			let items = menu.getElementsByClassName("selectStatusMenuItem");
			for (let item of items) {
				item.onclick = function(ev) {
					ev.stopPropagation();
					ev.preventDefault();
					let req = new XMLHttpRequest();
					let formData = new FormData();
					formData.append("path", sel.dataset.path);
					formData.append("name", "status");
					formData.append("value", item.dataset.value);
					req.open("post", "/api/update-property");
					req.send(formData);
					req.onload = function() {
						if (req.status == 200) {
							let oldClass = "statusDot-" + sel.dataset.entryType + "-" + sel.dataset.value;
							let newClass = "statusDot-" + sel.dataset.entryType + "-" + item.dataset.value;
							sel.classList.replace(oldClass, newClass);
							sel.dataset.value = item.dataset.value;
							menu.style.visibility = "hidden";
						} else {
							printErrorStatus(req.responseText);
						}
					}
					req.onerror = function(err) {
						printErrorStatus("network error occurred. please check whether the server is down.");
					}
				}
			}
		}
	}
	let thumbs = document.getElementsByClassName('thumbnail');
	for (let thumb of thumbs) {
		thumb.ondragover = function(event) {
			event.stopPropagation();
			event.preventDefault();
			event.currentTarget.classList.add("prepareDrop");
		}
		thumb.ondragleave = function(event) {
			event.stopPropagation();
			event.preventDefault();
			event.currentTarget.classList.remove("prepareDrop");
		}
		thumb.ondrop = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumbInput = event.currentTarget.getElementsByClassName("updateThumbnailInput")[0];
			thumbInput.files = event.dataTransfer.files;
			let thumb = parentWithClass(thumbInput, "thumbnail");
			updateThumbnail(thumb);
			event.currentTarget.classList.remove("prepareDrop");
		}
	}
	let thumbInputs = document.getElementsByClassName("updateThumbnailInput");
	for (let thumbInput of thumbInputs) {
		thumbInput.onchange = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = parentWithClass(thumbInput, "thumbnail");
			updateThumbnail(thumb);
		}
	}
	let delThumbButtons = document.getElementsByClassName("deleteThumbnailButton");
	for (let delButton of delThumbButtons) {
		delButton.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let thumb = parentWithClass(delButton, "thumbnail");
			deleteThumbnail(thumb);
		}
	}
	let assigneeInputs = document.getElementsByClassName("assigneeInput")
	for (let input of assigneeInputs) {
		let called = CalledByName[input.dataset.assignee];
		if (!called) {
			called = "";
		}
		input.value = called;
		input.dataset.oldValue = called;
		let oncomplete = function(value) {
			if (value == input.dataset.oldValue) {
				return;
			}
			let req = new XMLHttpRequest();
			let formData = new FormData();
			formData.append("path", input.dataset.path);
			formData.append("name", "assignee");
			formData.append("ctg", "property");
			formData.append("value", value);
			req.open("post", "/api/update-property");
			req.send(formData);
			req.onerror = function() {
				printErrorStatus("network error occurred. please check whether the server is down.");
			}
			req.onload = function() {
				if (req.status != 200) {
					printErrorStatus(req.responseText);
					return;
				}
				let called = CalledByName[value];
				if (!called) {
					called = "";
				}
				input.value = called;
				input.dataset.oldValue = called;
				if (!called) {
					printStatus("done");
					return;
				}
				// Give the assignee write permission of the entry.
				let r = new XMLHttpRequest();
				let data = new FormData();
				data.append("path", input.dataset.path);
				data.append("name", value);
				data.append("type", "user");
				data.append("value", "rw");
				r.open("post", "/api/add-or-update-access");
				r.send(data);
				r.onerror = function() {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
				r.onload = function() {
					if (r.status != 200) {
						printErrorStatus(r.responseText);
						return;
					}
					printStatus("done");
				}
			}
		}
		autoComplete(input, AllUserLabels, AllUserNames, oncomplete);
	}
	let infoContextMenuLoaders = document.getElementsByClassName("infoContextMenuLoader");
	for (let loader of infoContextMenuLoaders) {
		loader.onclick = function(event) {
			event.stopPropagation();
			event.preventDefault();
			let info = parentWithClass(loader, "info");
			if (info == null) {
				console.log("info not found");
				return;
			}
			let menu = document.getElementById("infoContextMenu");
			if (currentContextMenuLoader == loader) {
				currentContextMenuLoader = null;
				menu.style.visibility = "hidden";
				return;
			}
			let infoHistory = menu.getElementsByClassName("infoHistory")[0];
			infoHistory.href = "/logs?path=" + info.dataset.entry + "&category=" + info.dataset.infoType + "&name=" + info.dataset.infoName;
			let infoDelete = menu.getElementsByClassName("infoDelete")[0];
			infoDelete.onclick = function(ev) {
				ev.stopPropagation();
				ev.preventDefault();
				let req = new XMLHttpRequest();
				let formData = new FormData();
				formData.append("path", info.dataset.entry);
				formData.append("name", info.dataset.infoName);
				req.open("post", "/api/delete-" + info.dataset.infoType);
				req.send(formData);
				req.onload = function() {
					if (req.status == 200) {
						location.reload();
					} else {
						printErrorStatus(req.responseText);
					}
				}
				req.onerror = function(err) {
					printErrorStatus("network error occurred. please check whether the server is down.");
				}
			}
			let x = loader.offsetLeft;
			let y = loader.offsetTop + loader.offsetHeight;
			menu.style.visibility = "visible";
			menu.style.left = x + "px";
			menu.style.top = y + "px";
			currentContextMenuLoader = loader;
		}
	}
}

window.onpageshow = function() {
	let thumbnailImgs = document.getElementsByClassName("thumbnailImg");
	for (let img of thumbnailImgs) {
		if ((window.getComputedStyle(img).visibility == "visible") && (img.naturalWidth == 0)) {
			// TODO: It seems the following argument isn't true anymore. Please check again if we can delete this code.
			//
			// This means we uploaded the thumbnail, but doesn't show properly
			// as we arrive this page with browser's previous button.
			// Interestingly the "?t=curent-time" part are gone (at least) in firefox.
			// It makes the browser uses the old 'not found' cache.
			img.src = img.src.split("?")[0] + "?t=" + new Date().getTime();
		}
	}
}

function parentWithClass(from, clsName) {
	while (true) {
		let parent = from.parentElement;
		if (parent == null) {
			return null;
		}
		if (parent.classList.contains(clsName)) {
			return parent;
		}
		from = parent;
	}
}

function removeDragDropEvents(el) {
	el.ondragenter = null;
	el.ondragover = null;
	el.ondragleave = null;
	el.ondrop = null;
}

function updatePinnedPath(path, at) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("update_pinned_path", "1");
	formData.append("pinned_path", path);
	formData.append("pinned_path_at", at);
	req.open("post", "/api/update-user-setting");
	req.send(formData);
	req.onload = function() {
		if (req.status == 200) {
			location.reload();
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function updateQuickSearch(path, at) {
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("arrange_quick_search", "1");
	formData.append("quick_search_name", path);
	formData.append("quick_search_at", at);
	req.open("post", "/api/update-user-setting");
	req.send(formData);
	req.onload = function() {
		if (req.status == 200) {
			location.reload();
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function toggleRenameInput() {
	let input = document.getElementById("renameInput");
	if (input.style.display == "none") {
		input.style.display = "block";
		let end = input.value.length;
		input.setSelectionRange(0, end);
		input.focus();
	} else {
		input.style.display = "none";
	}
}

function updateThumbnail(thumb) {
	let img = thumb.getElementsByClassName("thumbnailImg")[0];
	let form = thumb.getElementsByClassName("updateThumbnailForm")[0];
	let now = new Date().getTime();
	if (thumb.dataset.lastUpload) {
		// Prevent Safari from firing this event twice.
		// TODO: resolve the base problem
		let last = Number(thumb.dataset.lastUpload);
		let d = now - last;
		if (d < 1000) {
			return;
		}
	}
	thumb.dataset.lastUpload = String(now);
	let req = new XMLHttpRequest();
	if (thumb.classList.contains("exists")) {
		form.action = form.action.replace("/api/add", "/api/update");
	} else {
		form.action = form.action.replace("/api/update", "/api/add");
	}
	req.open(form.method, form.action);
	req.send(new FormData(form));
	req.onload = function() {
		if (req.status == 200) {
			let entryPath = parentWithClass(thumb, "entry").dataset.entryPath;
			img.src = "/thumbnail" + entryPath + "?t=" + new Date().getTime();
			thumb.classList.remove("inherited");
			thumb.classList.add("exists");
			printStatus("done");
		} else {
			img.parentElement.style.border = "1px solid #D72";
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		img.parentElement.style.border = "1px solid #D72";
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function deleteThumbnail(thumb) {
	let img = thumb.getElementsByClassName("thumbnailImg")[0];
	let form = thumb.getElementsByClassName("deleteThumbnailForm")[0];
	let req = new XMLHttpRequest();
	req.open(form.method, form.action);
	req.send(new FormData(form));
	req.onload = function() {
		if (req.status == 200) {
			// the image is gone, reflect it to img tag (even if it will not visible).
			// TODO: inherit parent thumbnail
			img.src = img.src.split("?")[0] + "?t=" + new Date().getTime();
			form.action = form.action.replace("/api/update", "/api/add");
			thumb.classList.remove("exists");
			printStatus("done");
		} else {
			img.parentElement.style.border = "1px solid #D72";
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		img.parentElement.style.border = "1px solid #D72";
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
}

function resizeTextArea(textarea) {
	textarea.style.height = "auto";
	textarea.style.height = String(textarea.scrollHeight) + "px";
}

function submitUpdaterOrAdder(ev, input) {
	ev.stopPropagation();
	ev.preventDefault();
	let req = new XMLHttpRequest();
	let form = input.parentElement;
	let formData = new FormData(input.parentElement);
	let entPath = formData.get("path");
	let ctg = formData.get("ctg");
	let prop = formData.get("name");
	let marker = form.getElementsByClassName("updatingMarker")[0];
	req.onload = function() {
		marker.style.visibility = "hidden";
		if (req.status == 200) {
			// we know the value we just send,
			// but let's get the corrected value from server.
			let get = new XMLHttpRequest();
			let getFormData = new FormData();
			getFormData.append("path", entPath);
			getFormData.append("name", prop);
			get.open("post", "/api/get-" + ctg);
			get.onload = function() {
				if (get.status == 200) {
					let j = JSON.parse(get.responseText);
					if (j.Err != null) {
						printErrorStatus(j.Err);
						return;
					}
					let infoElem = document.querySelector(`[data-info-id="${entPath}.${ctg}.${prop}"]`);
					if (infoElem != null) {
						let valueElem = infoElem.querySelector(".itemValue");
						valueElem.innerText = j.Msg.Value;
						// remove possible 'invalid' class
						valueElem.classList.remove("invalid");

						// Look UpdatedAt to check it was actually updated.
						// It might not, if new value is same as the old one.
						let updated = new Date(j.Msg.UpdatedAt);
						let now = Date.now();
						let delta = (now - updated);
						let day = 24 * 60 * 60 * 100;
						if (delta <= day) {
							let dotElem = infoElem.querySelector(".recentlyUpdatedDot");
							dotElem.classList.add("unhide");
						}
					}
					printStatus("done");
					return;
				} else {
					printErrorStatus("update done, but failed to get the new value:" + get.responseText);
					return;
				}
			}
			get.send(getFormData);
		} else {
			printErrorStatus(req.responseText);
		}
	}
	req.onerror = function(err) {
		marker.style.visibility = "hidden";
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.open(form.method, form.action);
	req.send(formData);
	marker.style.visibility = "visible";
}

document.onkeydown = keyPressed;

function keyPressed(ev) {
	if (ev.code == "Escape") {
		// Will close floating UIs first, if any exists.
		let closed = false;
		let selectStatusMenus = document.getElementsByClassName("selectStatusMenu");
		for (let menu of selectStatusMenus) {
			if (menu.style.visibility != "hidden") {
				menu.style.visibility = "hidden";
				closed = true;
			}
		}
		let userMenu = document.getElementById("userAutoCompleteMenu");
		if (userMenu.style.visibility != "hidden") {
			userMenu.replaceChildren();
			userMenu.style.visibility = "hidden";
			closed = true;
		}
		let infoMenu = document.getElementById("infoContextMenu");
		if (infoMenu.style.visibility != "hidden") {
			infoMenu.style.visibility = "hidden";
			closed = true;
		}
		if (closed) {
			return;
		}
		// No float UIs were there. Do default job.
		toggleFooter();
	}
}

function hideAllItems() {
	let items = document.getElementById("entry-items");
	document.getElementById("property-box").classList.remove("selected");
	let props = items.getElementsByClassName("property-items");
	for (let p of props) {
		p.style.display = "none";
	}
	document.getElementById("environ-box").classList.remove("selected");
	let envs = items.getElementsByClassName("environ-items");
	for (let e of envs) {
		e.style.display = "none";
	}
	document.getElementById("access-box").classList.remove("selected");
	let accesses = items.getElementsByClassName("access-items");
	for (let a of accesses) {
		a.style.display = "none";
	}
}

function showItems(ctg) {
	let cls = document.getElementById(ctg + "-box").classList;
	let selected = cls.contains("selected")
	hideAllItems();
	if (selected) {
		return;
	}
	cls.add("selected");
	let items = document.getElementById("entry-items");
	let props = items.getElementsByClassName(ctg + "-items");
	for (let p of props) {
		p.style.display = "block";
	}
}

function showItemUpdater(entry, ctg, name, type, value) {
	showFooter();
	hideItemAdder();

	let updater = document.getElementById("itemUpdater");
	updater.style.display = "block";
	updater.getElementsByClassName("entryLabel")[0].innerText = entry;
	updater.getElementsByClassName("entryInput")[0].value = entry;
	updater.getElementsByClassName("categoryInput")[0].value = ctg;
	updater.getElementsByClassName("nameLabel")[0].innerText = name;
	updater.getElementsByClassName("nameInput")[0].value = name;
	updater.getElementsByClassName("typeInput")[0].value = type;
	updater.getElementsByClassName("valueForm")[0].action = "/api/update-" + ctg;
	updater.getElementsByTagName("button")[0].innerText = "Update";
	clearStatus();

	let valueInput = updater.getElementsByClassName("valueInput")[0];
	valueInput.placeholder = type;
	valueInput.value = value;
	resizeTextArea(valueInput);
	valueInput.focus();
}

function hideItemUpdater() {
	let updater = document.getElementById("itemUpdater");
	updater.style.display = "none";
}

let PropertyTypes = {{marshalJS $.PropertyTypes}}
let AccessorTypes = {{marshalJS $.AccessorTypes}}

function showItemAdder(entry, ctg) {
	// TODO: Add the item inplace?
	showFooter();
	hideItemUpdater();

	let adder = document.getElementById("itemAdder");
	adder.style.display = "block";
	adder.getElementsByClassName("entryLabel")[0].innerText = entry;
	adder.getElementsByClassName("entryInput")[0].value = entry;
	adder.getElementsByClassName("categoryInput")[0].value = ctg;
	adder.getElementsByTagName("button")[0].innerText = "Add";
	clearStatus();

	let nameInput = adder.getElementsByClassName("nameInput")[0]
	nameInput.value = name;
	nameInput.placeholder = ctg;
	let typeSel = adder.getElementsByClassName("typeSelect")[0]
	typeSel.innerHTML = "";
	let types = PropertyTypes;
	if (ctg == "access") {
		types = AccessorTypes;
	}
	for (let t of types) {
		let option = document.createElement("option");
		option.value = t;
		option.text = t;
		typeSel.appendChild(option)
	}
	adder.getElementsByClassName("valueForm")[0].action = "/api/add-" + ctg;

	let valueInput = adder.getElementsByClassName("valueInput")[0];
	valueInput.value = "";
	resizeTextArea(valueInput);

	nameInput.focus();
}

function hideItemAdder() {
	let adder = document.getElementById("itemAdder");
	adder.style.display = "none";
}

// showStatusBarOnly shows statusBar and hide other elements in footer. (need for eg. update thumbnail failed.)
// statusBar will be cleaned before it is shown.
function showStatusBarOnly() {
	showFooter();
	hideItemAdder();
	hideItemUpdater();
	let statusBar = document.getElementById("statusBar");
	statusBar.innerHTML = "";
}

function printStatus(s) {
	showStatusBarOnly();
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.remove("error");
	statusBar.innerHTML = s;
}

function printErrorStatus(e) {
	showStatusBarOnly();
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.add("error");
	statusBar.innerHTML = e;
}

function clearStatus() {
	let statusBar = document.getElementById("statusBar");
	statusBar.classList.remove("error");
	statusBar.innerHTML = "";
}

function toggleFooter() {
	let footer = document.getElementById("footer");
	if (footer.style.display == "block") {
		hideFooter();
	} else {
		showFooter();
	}
}

function showFooter() {
	let footer = document.getElementById("footer");
	footer.style.display = "block";
}

function hideFooter() {
	let footer = document.getElementById("footer");
	footer.style.display = "none";
}

function toggleCollapse() {
	let collapsible = this.parentElement;
	let content = collapsible.getElementsByClassName("content")[0];
	if (content.style.display == "none") {
		content.style.display = "block";
	} else {
		content.style.display = "none";
	}
}

var coll = document.getElementsByClassName("collapsible");
for (let i = 0; i < coll.length; i++) {
	coll[i].getElementsByClassName("title")[0].addEventListener("click", toggleCollapse);
}

function openDeleteEntryDialog(path) {
	// The dialog itself is not hidden but the parent sets the visibility.
	let dialogBg = document.getElementById("deleteEntryDialogBackground");
	let numSubEntries = -1;
	let req = new XMLHttpRequest();
	let formData = new FormData();
	formData.append("path", path);
	req.open("post", "/api/count-all-sub-entries");
	req.send(formData);
	req.onerror = function(err) {
		printErrorStatus("network error occurred. please check whether the server is down.");
	}
	req.onload = function() {
		if (req.status != 200) {
			printErrorStatus(req.responseText);
			return;
		}
		let j = JSON.parse(req.responseText);
		if (j.Err != null) {
			printErrorStatus(j.Err);
			return;
		}
		numSubEntries = j.Msg;
		if (numSubEntries != 0) {
			document.getElementById("deleteEntryDialogNoSub").style.display = "none";
			document.getElementById("deleteEntryDialogHasSub").style.display = "block";
			document.getElementById("deleteEntryDialogTotalSub").innerText = String(numSubEntries);
		} else {
			document.getElementById("deleteEntryDialogNoSub").style.display = "block";
			document.getElementById("deleteEntryDialogHasSub").style.display = "none";
			document.getElementById("deleteEntryDialogTotalSub").innerText = "";
		}
		document.getElementById("deleteEntryDialogEntry").innerText = path;
		dialogBg.style.visibility = "visible";
	}
	// cancel or confirm delete
	document.getElementById("cancelDeleteEntryButton").onclick = function() {
		dialogBg.style.visibility = "hidden";
	}
	document.getElementById("confirmDeleteEntryButton").onclick = function() {
		let req = new XMLHttpRequest();
		let formData = new FormData();
		formData.append("path", path);
		if (numSubEntries != 0) {
			formData.append("recursive", path);
		}
		req.open("post", "/api/delete-entry");
		req.send(formData);
		req.onerror = function(err) {
			printErrorStatus("network error occurred. please check whether the server is down.");
			dialogBg.style.visibility = "hidden";
		}
		req.onload = function() {
			if (req.status != 200) {
				printErrorStatus(req.responseText);
				dialogBg.style.visibility = "hidden";
				return;
			}
			let toks = path.split("/");
			toks.pop();
			let parent = toks.join("/");
			window.location.href = parent;
		}
	}
}

let AllUserNames = [
{{- range $u := $.AllUsers -}}
	"{{$u.Name}}",
{{end}}
];

let AllUserLabels = [
{{- range $u := $.AllUsers -}}
	"{{$u.Called}} ({{$u.Name}})",
{{end}}
];

// pun intended
let CalledByName = {
{{- range $u := $.AllUsers -}}
	"{{$u.Name}}": "{{$u.Called}}",
{{end}}
}

// autoComplete takes input tag and possible autocompleted values and label.
// It takes oncomplete function as an argument that will be called with user selected value.
// It will give oncomplete raw input value when it cannot find any item with the value.
function autoComplete(input, labels, vals, oncomplete) {
	// Turn off browser's default autocomplete behavior.
	input.setAttribute("autocomplete", "off");
	let focus = -1;
	input.oninput = function(event) {
		let search = input.value;
		if (search == "") {
			return;
		}
		let lsearch = search.toLowerCase();
		// reset focus on further input.
		focus = -1;
		let menu = document.getElementById("userAutoCompleteMenu");
		menu.style.visibility = "hidden";
		menu.style.left = String(input.offsetLeft) + "px";
		menu.style.top = String(input.offsetTop + input.offsetHeight) + "px";
		menu.replaceChildren();
		let items = [];
		for (let [i, l] of labels.entries()) {
			let ll = l.toLowerCase();
			let matchStart = ll.indexOf(lsearch);
			if (matchStart == -1) {
				continue
			}
			let matchEnd = matchStart + lsearch.length;
			let pre = l.slice(0, matchStart);
			let match = l.slice(matchStart, matchEnd);
			let post = l.slice(matchEnd, l.length);
			let item = document.createElement("div");
			item.classList.add("userAutoCompleteItem");
			item.innerHTML = pre + "<strong>" + match + "</strong>" + post;
			item.dataset.label = labels[i];
			item.dataset.value = vals[i];
			item.onclick = function(ev) {
				oncomplete(item.dataset.value);
				menu.replaceChildren();
				menu.style.visibility = "hidden";
				focus = -1;
			}
			menu.appendChild(item);
		}
		if (menu.children.length != 0) {
			menu.style.visibility = "visible";
		}
	}
	// Don't set input.onkeydown, it will swipe default (typing characters) behavior of input.
	input.addEventListener("keydown", function(event) {
		let menu = document.getElementById("userAutoCompleteMenu");
		let items = menu.getElementsByClassName("userAutoCompleteItem");
		if (event.key == "Tab") {
			// Let the cursor move to another input.
			menu.replaceChildren();
			menu.style.visibility = "hidden";
			return;
		}
		deactivate(items);
		if (event.key == "ArrowDown") {
			event.preventDefault();
			focus++;
			if (focus == items.length) {
				focus = -1;
			} else {
				activate(items, focus);
			}
		} else if (event.key == "ArrowUp") {
			event.preventDefault();
			if (focus == -1) {
				focus = items.length;
			}
			focus--;
			if (focus != -1) {
				activate(items, focus);
			}
		} else if (event.key == "Enter") {
			event.preventDefault();
			if (focus == -1) {
				if (items.length == 0) {
					oncomplete(input.value);
					menu.replaceChildren();
					menu.style.visibility = "hidden";
					focus = -1;
					return;
				}
				focus = 0;
			}
			items[focus].click();
		}
	})
	input.onkeyup = function(event) {
		let menu = document.getElementById("userAutoCompleteMenu");
		if (input.value == "") {
			menu.replaceChildren();
			menu.style.visibility = "hidden";
		}
	}
	function deactivate(items) {
		for (let item of items) {
			item.classList.remove("active");
		}
	}
	function activate(items, focus) {
		if (items.length == 0) {
			return;
		}
		if (focus == -1) {
			return;
		}
		items[focus].classList.add("active");
	}
}
