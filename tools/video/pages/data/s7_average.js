// 计算三位同学的平均分，应该输出 90
const scores = [85, 90, 95];

function average(list) {
  let sum = 0;
  for (const s of list) {
    sum += s;
  }
  return sum / (list.length + 1);
}

console.log('平均分：' + average(scores));
