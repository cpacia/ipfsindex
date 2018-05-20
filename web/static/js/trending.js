$(function(){
    $("#navSearch").removeClass("active");
    $("#navTrending").addClass("active");
    if (!more) {
        $('#nextPage').addClass("disabled");
    }
    if (page > 1) {
        $('#prevPage').removeClass("disabled");
    }
    if (selectedCategory != "") {
        $('#categoryMenuButton').html(selectedCategory);
    }
    $('#nextPage').click(function(){
        var selectedCategory = $('#categoryMenuButton').html();
        var catQuery = "";
        if (!selectedCategory.includes("Category")) {
            catQuery = "&category=" + selectedCategory;
        }
        if (more) {
            window.location = "/trending?page=" + (page + 1) + catQuery;
        }
    });
    $('#prevPage').click(function(){
        var selectedCategory = $('#categoryMenuButton').html();
        var catQuery = "";
        if (!selectedCategory.includes("Category")) {
            catQuery = "&category=" + selectedCategory;
        }
        if (page-1 > 0) {
            window.location = "/trending?page=" + (page - 1) + catQuery;
        }
    });
    $('.dropdown-toggle').dropdown();
    $('.categoryButton').click(function(event){
        window.location = "/trending?category=" + event.target.name;
    });
});

function goto(txid) {
    window.location = "/file/" + txid;
}