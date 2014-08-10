var LogObj = {
	level: [],
	tag: [],
	offset: 0,
	isLoading: false,


	reset: function() {
		this.level = [];
		this.tag = [];
		this.offset = 0;
		$("#cont").empty();
		this.buildUI()
	},

	resetViewsOffset: function() {
		this.offset = 0;
		$("#cont").empty();
		this.buildUI()
	},

	getUrl: function() {
		var url = '/db/logs/?offset=' + this.offset;
		if (this.tag.length > 0) {
			url += "&tag="+this.tag.join(",")
		}
		if (this.level.length > 0) {
			url += "&level="+this.level.join(",")
		}
		console.log(url)
		return url
	},

	addLogs: function() {
		var that = this;
		if (this.isLoading) {
			return;
		}
		NProgress.start();
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
			$("#cont").append(_.template($('#logs-list-template').html(), {logs: data}));
			that.isLoading = false;
			that.offset+=100;
			NProgress.done();
  		}).fail(function() {
  			that.isLoading = false;
  			NProgress.done()
  		});
	},

	activateScrollBinding: function() {
		var that = this;
		$("#releases").scroll(function() {
			if ($("#releases").scrollTop() > 300) {
				$("#navlinks").show()
			}else {
				$("#navlinks").hide()
			}

			if($("#releases").scrollTop() + $("#releases").height() == document.getElementById("releases").scrollHeight) {
				that.addLogs();
			}
		})
	},

	buildUI: function() {
		$("#ablevel").empty()
		$("#abtag").empty()
		var str = ""
		for(var i = 0; i < this.tag.length; i++) {
			str += "<button class='button-tag pure-button' onclick='app.removeTag(\""+this.tag[i]+"\")'>"
			str += this.tag[i]
			str +="</button>"
		}
		$("#abtag").append(str)

		var str2 = ""
		for(var i = 0; i < this.level.length; i++) {
			str2 += "<button class='button-tag pure-button' onclick='app.removeLevel(\""+this.level[i]+"\")'>"
			str2 += this.level[i]
			str2 +="</button>"
		}
		$("#ablevel").append(str2)

	},

	addTag: function(tag) {
		this.tag.push(tag)
		this.buildUI()
	},

	removeTag: function(tag) {
		for(var i = 0; i < this.tag.length;i++) {
			if (this.tag[i] == tag) {
				this.tag.splice(i, 1)
				break
			}
		}
		this.buildUI()
	},

	addLevel: function(level) {
		this.level.push(level)
		this.buildUI()
	},

	removeLevel: function(level) {
		for(var i = 0; i < this.level.length;i++) {
			if (this.level[i] == level) {
				this.level.splice(i, 1)
				break
			}
		}
		this.buildUI()
	}
	
}