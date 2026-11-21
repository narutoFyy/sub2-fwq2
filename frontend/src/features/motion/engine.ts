/** 这里静态引入，engine 只通过 loadMotion() 动态加载，Rollup 能按用到的函数摇树。 */
export { animate, inView, stagger } from 'motion-v'
