var LogObj = {
	level: "",
	offset: 0,
	isLoading: false,


	reset: function() {
		this.level = "";
		this.offset = 0;
		$("#lcont").empty();
	},

	addUI: function() {
		var that = this;
		$("#lcont").show();
		if (this.isLoading) {
			return;
		}
		if(document.getElementById("lcont").childNodes.length == 0) {
			var th
			$("#lcont").append(_.template($('#log-info-template').html()));
			$(".btn-group > button.btn").on("click", function(){
				var opt = this.innerHTML;
				if(opt === "clear") {
					that.clearLogs();
					return;
				}
				that.reset();
				if(opt === "all"){
					that.level = "";
				}else {
					that.level = opt.toUpperCase();
				}
				that.addUI();

			});
		}
		this.isLoading = true;
		var url = '/db/log/' + this.offset;
		url += (this.level == "") ? "/" : "/" + this.level;
		console.log(url);
		$.getJSON(url, function (data) {
			if ($.isEmptyObject(data)) {
				that.isLoading = false;
				toastr.info("no more data to display~~");
				return;
			}
			console.log(data);
			$("#ltbody").append(_.template($('#log-list-template').html(), {blogs: data}));
			that.isLoading = false;
			that.offset += 50;			  
		});
	},

	activateScrollBinding: function() {
		var that = this;
		$("#lcont").scroll(function() {
			if($("#lcont").scrollTop() + $("#lcont").height() == $("#lcont")[0].scrollHeight) {
				that.addUI();
			}
		})
	},

	clearLogs: function() {
		var that = this;
		$.post('/log/clearlogs', function(data) {
			//$("#tcont").html("");
			that.reset();
			that.addUI();
		});
	}
	
}