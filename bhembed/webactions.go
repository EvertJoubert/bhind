package bhembed

const webactionsjs string = `/*WebActions*/
function webactionRequestViaOptions(callbackurl,formid,actiontarget){
	postForm({
		url_ref:callbackurl,
		form_ref:formid,
		target:actiontarget
	});
}

function postForm(options){
	if(options==undefined) return;
	if(options.url_ref==undefined||options.url_ref==""){
		return;
	}
	if(options.form_ref==undefined||options.form_ref==""){
		return;
	}
	var progressElem="";
	var errorElem="";
	var urlref="";
	var formid="";
	var target="";

	if(options.progress_elem!=undefined){
		progressElem=options.progress_elem+"";
	} else{
		/*
		 * Default Block UI Progress
		 */
		$.blockUI({ 
			message : '<span style="font-size:1.2em" id="showprogress">Please wait ...</span>',
			css: { 
			border: 'none', 
	        padding: '15px', 
	        backgroundColor: '#000', 
	        '-webkit-border-radius': '10px', 
	        '-moz-border-radius': '10px', 
	        opacity: .7, 
	        color: '#fff'
			}
		});
		progressElem="#showprogress";
	}
	if(options.error_elem!=undefined){
		errorElem=options.error_elem+"";
	}	
	if(options.url_ref!=undefined){
		urlref=options.url_ref+"";
	}
	if(options.form_ref!=undefined){
		formid=options.form_ref+"";
	}
	if(options.target!=undefined){
		target=options.target+"";
	}
	
	var formData = new FormData();//new FormData($(formid)[0]);
	var formIds=formid.trim()==""?[]:formid.split("|")
	formIds.forEach(function(fid,i,arr){
		if($(fid).length){
			if(!$(fid).is("form")){
				$(fid+" :input").each(function(){
					 var input = $(this); // This is the jquery object of the input, do what you will
					 if(input.attr("name")!=""){
						 if(input.attr("type")!="button"&&input.attr("type")!="submit"&&input.attr("type")!="image"){
							 if(input.attr("type")=="file"){
								 formData.append(input.attr("name"),input[0].files[0]);
							 } else {
								 formData.append(input.attr("name"),input.val());
							 }
						 }
					 }
				});
				$(fid+" textarea").each(function(){
					var input = $(this);
					if(input.attr("name")!=""){
						formData.append(input.attr("name"),input.text());
					}
				});
			}
		}
	});
	
	var urlparams=getAllUrlParams(urlref);
	if (urlref.indexOf("?")>-1){
		urlref=urlref.slice(0,urlref.indexOf("?"));
	}
	if (urlparams!=undefined){
		Object.keys(urlparams).forEach(function(key) {
			if(Array.isArray(urlparams[key])){
				urlparams[key].forEach(function(val){
					formData.append(key,val);
				});
			} else {
				formData.append(key,urlparams[key]);
			}
		});
	}
	
	$.ajax({
        xhr: function () {
            var xhr = $.ajaxSettings.xhr();
            xhr.upload.onprogress = function (e) {
            	if(progressElem!=undefined&&progressElem!=""){
            		$(progressElem).html(Math.floor(e.loaded / e.total * 100) + '%');
            	}
            };
            return xhr;
        },
        contentType: false,
        processData: false,
        type: 'POST',
        data: formData,
        url: urlref,
        success: function (response) {
        	if(progressElem!=undefined&&progressElem=="#showprogress"){
        		$.unblockUI();
        	}
        	var parsed=parseActiveString("script||","||script",response);
        	var parsedScript=parsed[1].join("");
        	response=parsed[0].trim();
        	
        	var targets=[];
        	var targetSections=[];
        	if(response!=""){
        		if(response.indexOf("replace-content||")>-1){
        			parsed=parseActiveString("replace-content||","||replace-content",response);
        			response=parsed[0];
        			
        			parsed[1].forEach(function(possibleTargetContent,i){
        				if(possibleTargetContent.indexOf("||")>-1){
        					targets[targets.length]=[possibleTargetContent.substring(0,possibleTargetContent.indexOf("||")),possibleTargetContent.substring(possibleTargetContent.indexOf("||")+"||".length,possibleTargetContent.length)];
        				}        				
        			});
        		}
        		targets.unshift([target,response]);
        	}
        	if(targets.length>0){
        		targets.forEach(function(targetSec){
        			if ($(targetSec[0]).length>0) {
        				$(targetSec[0]).html(targetSec[1]);
        			}
        		});
        	}
        	if(parsedScript!=""){
        		eval(parsedScript);
        	}
        },
        error: function(jqXHR, textStatus) {
        	if(progressElem!=undefined&&progressElem=="#showprogress"){
        		$.unblockUI();
        	}
        	if(errorElem!=undefined&&errorElem!=""){
        		$(errorElem).html("Error loading request: "+textStatus);
        	}
        }
    });

}

function parseActiveString(labelStart,labelEnd,passiveString){
	this.parsedPassiveString="";
	this.parsedActiveString="";
	this.parsedActiveArr=[];
	this.passiveStringIndex=0;
	this.passiveStringArr=Array.from(passiveString);
	
	this.passiveStringLen=this.passiveStringArr.length;
	
	this.labelStartIndex=0;
	this.labelEndIndex=0;
	
	this.labelStartArr=Array.from(labelStart);
	this.labelEndArr=Array.from(labelEnd);
	this.pc='';
	
	this.passiveStringArr.forEach(function(c,i){
		
		if(this.labelEndIndex==0&&this.labelStartIndex<this.labelStartArr.length){
			if(this.labelStartIndex>0&&this.labelStartArr[this.labelStartIndex-1]==pc&&this.labelStartArr[this.labelStartIndex]!=c){
				this.parsedPassiveString+=labelStart.substring(0,this.labelStartIndex);
				this.labelStartIndex=0;
			}
			if(this.labelStartArr[this.labelStartIndex]==c){
				
				this.labelStartIndex++;
				if(this.labelStartIndex==this.labelStartArr.length){
					
				}
			}
			else{
				if(this.labelStartindex>0){
					this.parsedPassiveString+=labelStart.substring(0,this.labelStartIndex);
					this.labelStartIndex=0;
				}
				this.parsedPassiveString+=(c+"");
			}
		}
		else if(this.labelStartIndex==this.labelStartArr.length&&this.labelEndIndex<this.labelEndArr.length){
			if(this.labelEndArr[this.labelEndIndex]==c){
				this.labelEndIndex++;
				if(this.labelEndIndex==this.labelEndArr.length){
					this.parsedActiveArr[this.parsedActiveArr.length]=this.parsedActiveString+"";
					this.parsedActiveString="";
					this.labelEndIndex=0;
					this.labelStartIndex=0;
				}
			}
			else{
				if(this.labelEndIndex>0){
					this.parsedActiveString+=labelEnd.substring(0,this.labelEndIndex);
					this.labelEndIndex=0;
				}
				this.parsedActiveString+=(c+"");
			}
		}
		
		this.pc=c;		
	});
	return [this.parsedPassiveString,this.parsedActiveArr]
}

$(function () {
	
});
`
