var cmd = "";
var offset = 0;
var loading = false;
$(window).load(function() {
	document.onmousewheel = moveObject;
	addUI();
	$("#clear").click(function(e) {
		e.preventDefault();
		$.post('/log/clearlogs', function(data) {
			$("#tcont").html("");
			addUI();
		});
	})
	$("#all").click(function(e) {
		if(cmd == "")
			return
		cmd = "";
		offset = 0;
		$("#tcont").html("");
		addUI();
	})
	$("#info").click(function(e) {
		if(cmd == "INFO")
			return
		cmd = "INFO";
		offset = 0;
		$("#tcont").html("");
		addUI();
	})
	$("#error").click(function(e) {
		if(cmd == "ERROR")
			return
		offset = 0;
		cmd = "ERROR";
		$("#tcont").html("");
		addUI();
	})
});

function addUI() {
	if(loading == true)
		return;
	loading = true;
	url = "/log/";
	url += offset;
	if(cmd != "") {
		url += "/"+cmd;
	}
	$.getJSON(url, function(data) {
		if($.isEmptyObject(data)) {
			loading = false;
			return;
		}
		$.each(data, function(key, val) {
		  	$("<div class=\"row test\"><div class=\"col-lg-2\">"+val.Date+" "+val.Timestamp+"</div><div class=\"col-lg-1\">"+val.Line+"</div><div class=\"col-lg-1\">"+val.Lvl+"</div><div class=\"col-lg-7\">"+val.Message+"</div></div>").appendTo("#tcont");
		  	if(data.length-1 == key) {
		  		loading = false;
		  		offset += 50;
		  	}
		});			  
	});
}

function moveObject(event) {
	var delta = 0;
		 
    if (!event) event = window.event;
    	// normalize the delta
	if (event.wheelDelta) {
		delta = event.wheelDelta / 60; 
	} else if (event.detail) {
		delta = -event.detail / 2;
	}
	var currPos=document.getElementById('tcont').offsetTop;
	var amount2 = currPos *-1;
	console.log(currPos);
	console.log(amount2);

	if(amount2 > $("#tcont").outerHeight()-1400 ) {
		console.log("addUI()");
		addUI();
	}

	if(delta < 0) 
		currPos=parseInt(currPos)-(128+10);
	else 
		currPos=parseInt(currPos)+(128+10);

	if(currPos > 0)
		currPos = 0;
	//moving the position of the object
	document.getElementById('tcont').style.top = currPos+"px";
}