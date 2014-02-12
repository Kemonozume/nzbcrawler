var AppObj = {
	name: "none",
	genre: "none",
	offset: 0,
	isLoading: false,
	old_info: undefined,


	reset: function() {
		this.name = "none";
		this.genre = "none";
		this.offset = 0;
		this.old_info = undefined;
		$(tcont).empty();
	},

	addUI: function() {
		$("#tcont").show();
		if (this.isLoading) {
			return;
		}
		this.isLoading = true;
		var url = '/db/events/' + this.offset + "/" + this.genre + "/" + this.name;
		var that = this;
		$.getJSON(url, function (data) {
			if ($.isEmptyObject(data)) {
				that.isLoading = false;
				if(document.getElementById("tcont").childNodes.length == 0) {
					toastr.error("kein ergebniss gefunden");
				}else {
					toastr.info("no more data to display~~");
				}
				
				return;
			}
			console.log(data);
			$("#tcont").append(_.template($('#release-list-template').html(), {releases: data}));
			that.isLoading = false;
			that.offset+=200;
		});
	},

	activateScrollBinding: function() {
		var that = this;
		$(tcont).scroll(function() {
			if($(tcont).scrollTop() + $(tcont).height() == tcont.scrollHeight) {
				that.addUI();
			}
		})
	},

	showInfo: function(view) {
		var info = view.parentNode.childNodes[3];

	    if(this.old_info == undefined) {
	        this.old_info = info;
	    }

	    if(this.old_info != info) {
	        $(this.old_info.parentNode.nextElementSibling).show();
	        $(this.old_info.parentNode.nextElementSibling.nextElementSibling).show();
	        $(this.old_info).hide();
	    }

	    this.old_info = info;

	    if(info.style.display == "none" || info.style.display == "") {
	        $(this.old_info.parentNode.nextElementSibling).hide();
	        $(this.old_info.parentNode.nextElementSibling.nextElementSibling).hide();
	        $(this.old_info).show();
	    }
	    else {
	        $(this.old_info.parentNode.nextElementSibling).show();
	        $(this.old_info.parentNode.nextElementSibling.nextElementSibling).show();
	        $(this.old_info).hide();
	    }
	    
	},

	setTag: function(genre) {
    	parent.location.hash = "/search";
		this.reset();
		this.genre = genre;
		document.getElementById("s-genre").value = genre;
		this.addUI();
	}


	
}