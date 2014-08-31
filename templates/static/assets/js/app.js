var AppObj = {
	name: "",
	tags: [],
	offset: 0,
	isLoading: false,


	reset: function() {
		this.name = "";
		this.tags = [];
		this.offset = 0;
		$("#cont").empty();
		this.buildUI()
	},

	resetViewsOffset: function() {
		this.offset = 0;
		$("#cont").empty();
		this.buildUI()
	},

	addReleases: function() {
		if (this.isLoading) {
			return;
		}
		NProgress.start();
		this.isLoading = true;
		var that = this;
		$.ajax({
  			url: this.getUrl(),
  			dataType: "json",
  		}).done(function(data) {
  			if ($.isEmptyObject(data)) {
				that.isLoading = false;
				NProgress.done();
				return;
			}
			$("#cont").append(_.template($('#release-list-template').html(), {releases: data}));
			that.isLoading = false;
			that.offset+=100;
			NProgress.done();
  		}).fail(function() {
  			that.isLoading = false;
  			NProgress.done()
  		});
		
	},

	getUrl: function() {
		var url = '/db/releases/?offset=' + this.offset;
		if (this.tags.length > 0) {
			url += "&tags="+this.tags.join(",")
		}
		if (this.name != "") {
			url += "&name="+this.name
		}
		console.log(url)
		return url
	},

	activateScrollBinding: function() {
		var that = this;

		var bod = document.getElementsByTagName("body")[0];

		$(window).scroll(function() {
			if($(window).scrollTop() + $(window).height() == bod.scrollHeight) {
				that.addReleases();
			}
		})

	},

	buildUI: function() {
		$("#abtags").empty()
		$("#abname").empty()
		var str = ""
		for(var i = 0; i < this.tags.length; i++) {
			str += "<button class='btn btn-default btn-xs btn-flat btn-rem' onclick='app.removeTag(\""+this.tags[i]+"\")'>"
			str += this.tags[i]
			str +="</button>"
		}
		$("#abtags").append(str)
		if (this.name != "") {
			$("#abname").append("<button class='btn btn-default btn-xs btn-flat btn-rem' onclick='app.removeName()'>"+this.name+"</button>")
		}
	},

	addTag: function(tag) {
		this.tags.push(tag)
		this.buildUI()
	},

	removeTag: function(tag) {
		for(var i = 0; i < this.tags.length;i++) {
			if (this.tags[i] == tag) {
				this.tags.splice(i, 1)
				break
			}
		}
		this.buildUI()
	},

	addName: function(name) {
		this.name = name
		this.buildUI()
	},

	removeName: function() {
		this.name = ""
		this.buildUI()
	}
	
}