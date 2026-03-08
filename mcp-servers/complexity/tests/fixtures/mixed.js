function simpleGreet(name) {
    return "Hello " + name;
}

function processData(items, filter, transform) {
    let results = [];
    for (let i = 0; i < items.length; i++) {
        if (filter(items[i])) {
            if (transform) {
                results.push(transform(items[i]));
            } else {
                results.push(items[i]);
            }
        } else if (items[i] !== null && items[i] !== undefined) {
            results.push(items[i].toString());
        }
    }
    return results;
}
