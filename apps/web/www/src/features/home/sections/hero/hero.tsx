import React, { useState,useRef,useEffect } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { ChevronLeft, ChevronRight, Database, Zap, Bell, ShieldCheck } from 'lucide-react';


interface ElasticHueSliderProps {
  value: number;
  onChange: (value: number) => void;
  min?: number;
  max?: number;
  step?: number;
  label?: string;
}

const ElasticHueSlider: React.FC<ElasticHueSliderProps> = ({
  value,
  onChange,
  min = 0,
  max = 360,
  step = 1,
  label = 'Adjust Hue',
}) => {
  const [isDragging, setIsDragging] = useState(false);
  const sliderRef = useRef<HTMLDivElement>(null);

  const progress = ((value - min) / (max - min));
  const thumbPosition = progress * 100; // Percentage

  const handleMouseDown = () => setIsDragging(true);
  const handleMouseUp = () => setIsDragging(false);

  // We use the native input for actual dragging and value updates
  // and overlay custom elements for styling and animation.

  return (
    <div className="scale-50 relative w-full max-w-xs flex flex-col items-center" ref={sliderRef}>
      {label && <label htmlFor="hue-slider-native" className="text-gray-300 text-sm mb-1">{label}</label>}
      <div className="relative w-full h-5 flex items-center"> {/* Wrapper for track and thumb */}
        {/* Native input: Handles interaction, but visually hidden */}
        <input
          id="hue-slider-native"
          type="range"
          min={min}
          max={max}
          step={step}
          value={value}
          onChange={(e) => onChange(Number(e.target.value))}
          onMouseDown={handleMouseDown}
          onMouseUp={handleMouseUp}
          onTouchStart={handleMouseDown}
          onTouchEnd={handleMouseUp}
          // Style to make it cover the custom track but be transparent
          className="absolute inset-0 w-full h-full appearance-none bg-transparent cursor-pointer z-20"
          style={{ WebkitAppearance: 'none' /* For Safari */ }}
        />

        {/* Custom Track */}
        <div className="absolute left-0 w-full h-1 bg-gray-700 rounded-full z-0"></div>

        {/* Custom Fill (Optional but nice visual) */}
        <div
            className="absolute left-0 h-1 bg-blue-500 rounded-full z-10"
            style={{ width: `${thumbPosition}%` }}
        ></div>


        {/* Custom Thumb (Animated) */}
        {/* Position the thumb wrapper based on progress, then center the thumb inside */}
        <motion.div
          className="absolute top-1/2 transform -translate-y-1/2 z-30"
          style={{ left: `${thumbPosition}%` }}
          // Animate scale based on dragging state
          animate={{ scale: isDragging ? 1.2 : 1 }}
          transition={{ type: "spring", stiffness: 500, damping: isDragging ? 20 : 30 }} // Springy animation
        >
           
        </motion.div>
      </div>

       {/* Optional: Display current value below */}
       <AnimatePresence mode="wait">
         <motion.div
           key={value} // Key changes when value changes, triggering exit/enter
           initial={{ opacity: 0, y: -5 }}
           animate={{ opacity: 1, y: 0 }}
           exit={{ opacity: 0, y: 5 }}
           transition={{ duration: 0.2 }}
           className="text-xs text-gray-500 mt-2" // Increased margin top for spacing
         >
           {value}°
         </motion.div>
       </AnimatePresence>
    </div>
  );
};

interface FeatureItemProps {
  name: string;
  value: string;
  position: string;
}

interface LightningProps {
  hue?: number;
  xOffset?: number;
  speed?: number;
  intensity?: number;
  size?: number;
}

const Lightning: React.FC<LightningProps> = ({
  hue = 230,
  xOffset = 0,
  speed = 1,
  intensity = 1,
  size = 1,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const resizeCanvas = () => {
      canvas.width = canvas.clientWidth;
      canvas.height = canvas.clientHeight;
    };
    resizeCanvas();
    window.addEventListener("resize", resizeCanvas);

    const gl = canvas.getContext("webgl");
    if (!gl) {
      console.error("WebGL not supported");
      return;
    }

    const vertexShaderSource = `
      attribute vec2 aPosition;
      void main() {
        gl_Position = vec4(aPosition, 0.0, 1.0);
      }
    `;

    const fragmentShaderSource = `
      precision mediump float;
      uniform vec2 iResolution;
      uniform float iTime;
      uniform float uHue;
      uniform float uXOffset;
      uniform float uSpeed;
      uniform float uIntensity;
      uniform float uSize;
      
      #define OCTAVE_COUNT 10

      // Convert HSV to RGB.
      vec3 hsv2rgb(vec3 c) {
          vec3 rgb = clamp(abs(mod(c.x * 6.0 + vec3(0.0,4.0,2.0), 6.0) - 3.0) - 1.0, 0.0, 1.0);
          return c.z * mix(vec3(1.0), rgb, c.y);
      }

      float hash11(float p) {
          p = fract(p * .1031);
          p *= p + 33.33;
          p *= p + p;
          return fract(p);
      }

      float hash12(vec2 p) {
          vec3 p3 = fract(vec3(p.xyx) * .1031);
          p3 += dot(p3, p3.yzx + 33.33);
          return fract((p3.x + p3.y) * p3.z);
      }

      mat2 rotate2d(float theta) {
          float c = cos(theta);
          float s = sin(theta);
          return mat2(c, -s, s, c);
      }

      float noise(vec2 p) {
          vec2 ip = floor(p);
          vec2 fp = fract(p);
          float a = hash12(ip);
          float b = hash12(ip + vec2(1.0, 0.0));
          float c = hash12(ip + vec2(0.0, 1.0));
          float d = hash12(ip + vec2(1.0, 1.0));
          
          vec2 t = smoothstep(0.0, 1.0, fp);
          return mix(mix(a, b, t.x), mix(c, d, t.x), t.y);
      }

      float fbm(vec2 p) {
          float value = 0.0;
          float amplitude = 0.5;
          for (int i = 0; i < OCTAVE_COUNT; ++i) {
              value += amplitude * noise(p);
              p *= rotate2d(0.45);
              p *= 2.0;
              amplitude *= 0.5;
          }
          return value;
      }

      void mainImage( out vec4 fragColor, in vec2 fragCoord ) {
          // Normalized pixel coordinates.
          vec2 uv = fragCoord / iResolution.xy;
          uv = 2.0 * uv - 1.0;
          uv.x *= iResolution.x / iResolution.y;
          // Apply horizontal offset.
          uv.x += uXOffset;
          
          // Adjust uv based on size and animate with speed.
          uv += 2.0 * fbm(uv * uSize + 0.8 * iTime * uSpeed) - 1.0;
          
          float dist = abs(uv.x);
          // Compute base color using hue.
          vec3 baseColor = hsv2rgb(vec3(uHue / 360.0, 0.7, 0.8));
          // Compute color with intensity and speed affecting time.
          vec3 col = baseColor * pow(mix(0.0, 0.07, hash11(iTime * uSpeed)) / dist, 1.0) * uIntensity;
          col = pow(col, vec3(1.0));
          fragColor = vec4(col, 1.0);
      }

      void main() {
          mainImage(gl_FragColor, gl_FragCoord.xy);
      }
    `;

    const compileShader = (
      source: string,
      type: number
    ): WebGLShader | null => {
      const shader = gl.createShader(type);
      if (!shader) return null;
      gl.shaderSource(shader, source);
      gl.compileShader(shader);
      if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
        console.error("Shader compile error:", gl.getShaderInfoLog(shader));
        gl.deleteShader(shader);
        return null;
      }
      return shader;
    };

    const vertexShader = compileShader(vertexShaderSource, gl.VERTEX_SHADER);
    const fragmentShader = compileShader(
      fragmentShaderSource,
      gl.FRAGMENT_SHADER
    );
    if (!vertexShader || !fragmentShader) return;

    const program = gl.createProgram();
    if (!program) return;
    gl.attachShader(program, vertexShader);
    gl.attachShader(program, fragmentShader);
    gl.linkProgram(program);
    if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
      console.error("Program linking error:", gl.getProgramInfoLog(program));
      return;
    }
    gl.useProgram(program);

    const vertices = new Float32Array([
      -1, -1, 1, -1, -1, 1, -1, 1, 1, -1, 1, 1,
    ]);
    const vertexBuffer = gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, vertexBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, vertices, gl.STATIC_DRAW);

    const aPosition = gl.getAttribLocation(program, "aPosition");
    gl.enableVertexAttribArray(aPosition);
    gl.vertexAttribPointer(aPosition, 2, gl.FLOAT, false, 0, 0);

    const iResolutionLocation = gl.getUniformLocation(program, "iResolution");
    const iTimeLocation = gl.getUniformLocation(program, "iTime");
    const uHueLocation = gl.getUniformLocation(program, "uHue");
    const uXOffsetLocation = gl.getUniformLocation(program, "uXOffset");
    const uSpeedLocation = gl.getUniformLocation(program, "uSpeed");
    const uIntensityLocation = gl.getUniformLocation(program, "uIntensity");
    const uSizeLocation = gl.getUniformLocation(program, "uSize");

    const startTime = performance.now();
    const render = () => {
      resizeCanvas();
      gl.viewport(0, 0, canvas.width, canvas.height);
      gl.uniform2f(iResolutionLocation, canvas.width, canvas.height);
      const currentTime = performance.now();
      gl.uniform1f(iTimeLocation, (currentTime - startTime) / 1000.0);
      gl.uniform1f(uHueLocation, hue);
      gl.uniform1f(uXOffsetLocation, xOffset);
      gl.uniform1f(uSpeedLocation, speed);
      gl.uniform1f(uIntensityLocation, intensity);
      gl.uniform1f(uSizeLocation, size);
      gl.drawArrays(gl.TRIANGLES, 0, 6);
      requestAnimationFrame(render);
    };
    requestAnimationFrame(render);

    return () => {
      window.removeEventListener("resize", resizeCanvas);
    };
  }, [hue, xOffset, speed, intensity, size]);

  return <canvas ref={canvasRef} className="w-full h-full relative" />;
};


interface FeatureItemProps {
  name: string;
  value: string;
  position: string;
  icon: React.ReactNode;
}

const FeatureItem: React.FC<FeatureItemProps> = ({ name, value, position, icon }) => {
  return (
    <div className={`relative md:absolute ${position} z-10 group transition-all duration-500 w-full md:w-auto`}>
      <motion.div 
        whileHover={{ scale: 1.02, y: -2 }}
        className="relative flex items-center gap-3 p-3 md:p-4 rounded-2xl bg-white/5 backdrop-blur-md border border-white/10 shadow-2xl overflow-hidden"
      >
        {/* Animated Background Gradient on Hover */}
        <div className="absolute inset-0 bg-gradient-to-br from-purple-500/10 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
        
        {/* Icon Container */}
        <div className="relative flex items-center justify-center w-8 h-8 md:w-10 md:h-10 rounded-xl bg-white/10 group-hover:bg-purple-500/20 transition-colors duration-500">
          <div className="text-white/80 group-hover:text-white transition-colors duration-500">
            {icon}
          </div>
        </div>

        {/* Text Content */}
        <div className="relative flex flex-col">
          <span className="font-semibold text-sm md:text-base text-white/90 group-hover:text-white transition-colors duration-500 whitespace-nowrap">
            {name}
          </span>
          {value && (
            <span className="text-[10px] md:text-xs text-white/50 group-hover:text-white/70 transition-colors duration-500 whitespace-nowrap">
              {value}
            </span>
          )}
        </div>

        {/* Glow Effect */}
        <div className="absolute -inset-2 bg-purple-500/5 blur-2xl opacity-0 group-hover:opacity-100 transition-opacity duration-500 -z-10" />
      </motion.div>
    </div>
  );
};

export const HeroSection: React.FC = () => {
  // State for the lightning hue
  const [lightningHue, setLightningHue] = useState(245); // Default hue

  const containerVariants = {
    hidden: { opacity: 0 },
    visible: {
      opacity: 1,
      transition: {
        staggerChildren: 0.3,
        delayChildren: 0.2
      }
    }
  };

  const itemVariants = {
    hidden: { y: 20, opacity: 0 },
    visible: {
      y: 0,
      opacity: 1,
      transition: {
        duration: 0.5,
        ease: "easeOut"
      }
    }
  };

  return (
    <div className="relative w-full bg-black text-white overflow-hidden">
      {/* Main container with space for content */}
      <div className="relative z-20 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 h-screen">

        {/* Main hero content */}
        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="visible"
          className="mt-24 relative z-30 flex flex-col items-center text-center max-w-4xl mx-auto "
        > 
          <motion.button
            variants={itemVariants}
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            className="flex items-center space-x-2 px-4 py-2 bg-white/5 hover:bg-white/10 backdrop-blur-sm rounded-full text-sm mb-6 transition-all duration-300 group" // Reduced mb slightly
          >
            <span>Download the Desktop App</span>
            <ChevronRight className='size-4 transform group-hover:translate-x-1 transition-transform duration-300' />
          </motion.button>

          <motion.h1
            variants={itemVariants}
            className="text-5xl md:text-7xl font-light mb-2"
          >
            The <strong>Best</strong> Free LocalStack Alternative
          </motion.h1>

          <motion.h2
            variants={itemVariants}
            className="text-3xl md:text-5xl pb-3 font-light bg-gradient-to-r from-gray-100 via-gray-200 to-gray-300 bg-clip-text text-transparent"
          >
            AWS Cloud, <strong>100% Local</strong>
          </motion.h2>

          {/* Description */}
          <motion.p
            variants={itemVariants}
            className="text-gray-400 mb-9 max-w-2xl"
          >
            MildStack emulates the AWS API on your machine. Build and test with S3, DynamoDB, Lambda, SQS, SNS, and EventBridge without cloud complexity or costs.
          </motion.p>


    
        </motion.div>

        {/* Floating Widgets / Features */}
        <motion.div
          variants={containerVariants}
          initial="hidden"
          animate="visible"
          className="grid grid-cols-1 sm:grid-cols-2 md:block gap-4 w-full z-[200] relative md:absolute md:top-[45%] mt-12 md:mt-0 px-4 md:px-0"
        >
          <motion.div variants={itemVariants}>
            <FeatureItem icon={<Database className="size-5" />} name="S3 & DynamoDB" value="Fast local storage" position="md:left-0 lg:left-10 md:top-40" />
          </motion.div>
          <motion.div variants={itemVariants}>
            <FeatureItem icon={<Zap className="size-5" />} name="Lambda & SQS" value="Serverless & Queues" position="md:left-1/4 md:top-24" />
          </motion.div>
          <motion.div variants={itemVariants}>
            <FeatureItem icon={<Bell className="size-5" />} name="SNS & EventBridge" value="Pub/Sub & Events" position="md:right-1/4 md:top-24" />
          </motion.div>
          <motion.div variants={itemVariants}>
            <FeatureItem icon={<ShieldCheck className="size-5" />} name="100% Local" value="No cloud costs" position="md:right-0 lg:right-10 md:top-40" />
          </motion.div>
        </motion.div>
      </div>

      {/* Background elements */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 1 }}
        className="absolute inset-0 z-0"
      >
        {/* Dark overlay */}
        <div className="absolute inset-0 bg-black/80"></div>

        {/* Glowing circle */}
        <div className="absolute top-[55%] left-1/2 transform -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] rounded-full bg-gradient-to-b from-purple-500/20 to-pink-600/10 blur-3xl"></div>

        {/* Central light beam - now using the state variable for hue */}
        <div className="absolute top-0 w-[100%] left-1/2 transform -translate-x-1/2 h-full">
          <Lightning
            hue={lightningHue} // Use the state variable here
            xOffset={0}
            speed={1.6}
            intensity={0.6}
            size={2}
          />
        </div>

        {/* Planet/sphere */}
        <div className="z-10 absolute top-[55%] left-1/2 transform -translate-x-1/2 w-[600px] h-[600px] backdrop-blur-3xl rounded-full bg-[radial-gradient(circle_at_25%_90%,_#7F00FF_15%,_#000000de_70%,_#000000ed_100%)]"></div>
      </motion.div>
    </div>
  );
};
