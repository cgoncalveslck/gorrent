/**
 * @param {string} word
 * @return {number}
 */
var minimumPushes = function(word) {
  const keys = [
    [],[],[],
    [],[],[],
    [],[],[],
  ]

  let nextKey = () => {
    let key = 0;
    for (let i = 0; i < keys.length; i++) {
      if (keys[i].length < keys[key].length) {
        key = i;
      }
    }
    return key;
  }
  let biggestPress = 0
  for (let i = 0; i < word.length; i++) {
    const char = word[i];
  
    keys[nextKey].push(char);
    nextKey++;
  }
  
  return keys
};

console.log(minimumPushes('abc')); // 3