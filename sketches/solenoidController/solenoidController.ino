/*
JAF 2016-05-07. Solenoid Driver controller.
 Uses I2C to control a Gameplan SDU board.
 
 J? 
 1 = Relay Control btw J3-12 / J1-21
 2 = Relay Control btw Solenoid Grounds and J2-12 to 15
 3 = Q18 Driver J2-10 (to Ground) and J3-20 (to Ground)
 4 = Q8 Driver J2-9 (to Ground) and J3-19 (to Ground)
 5 = Sol Adr 1
 6 = Sol Adr 2
 7 = Key
 8 = Sol Adr 3
 9 = Sol Adr 4
 
 Low PortB (Active High):
 0=sol Address 0 (to J?-5)
 1=sol Address 1 (to J?-6)
 2=sol Address 2 (to J?-8)
 3=sol Address 3 (to J?-9)
 
 Low PortC (Active High):
 0=Relay Control J3-12/J1-21 (to J?-1)
 1=Relay Control Grounds to J2-12 to 15
 2=Q18 Driver (to J?-3)
 3=Q8 Driver (to J?-4)
 
 
 PortB = ####1111 turns off all solenoids.
 
 */

#include <avr/io.h>
#include <Wire.h>
#define F_CPU 16000000UL  //16mhz clock
#include <util/delay.h> 
#define SOLENOID_ADDRESS 0x03 //i2c address here. = 0x03

// Pin 13 has an LED connected on most Arduino boards.
// PB5 is the LED
// give it a name:
int led = 13; 

unsigned char _slowBlink;
unsigned char _fastBlink;
int _timerControl;

//holds the status of the lamp.
//00=off, 01=on, 10=slow blink, 11=fast blink
unsigned char _solenoidControl[17]; 
int _lastSolenoid; //last Solenoid ID command sent. Makes debugging easier

// the setup routine runs once when you press reset:
void setup() {                
  // initialize the digital pin as an output.
  pinMode(led, OUTPUT);     
  Serial.begin(9600); // start serial for output. Should only have this if debugging
  _lastSolenoid = 0;

  ClearAllSolenoids();

  DDRC = DDRC | B00001111; //Port C0-C3 output for Lamp Strobe0-Strobe3
  DDRB = DDRB | B00001111; //Port B0-B3 output for Lamp0-Lamp3

  _timerControl = 0;

  // initialize timer1 
  noInterrupts();           // disable all interrupts
  TCCR1A = 0;
  TCCR1B = 0;
  TCNT1  = 0;

  OCR1A =  15625; //31250;            // compare match register 16MHz/256/2Hz (4 hz)
  TCCR1B |= (1 << WGM12);   // CTC mode
  TCCR1B |= (1 << CS12);    // 256 prescaler 
  TIMSK1 |= (1 << OCIE1A);  // enable timer compare interrupt
  interrupts();             // enable all interrupts

    Wire.begin(SOLENOID_ADDRESS);                // join i2c bus 
  // register i2 events
  Wire.onReceive(receiveEvent); 
  Wire.onRequest(sendEvent);

  Serial.println("Initialized!");
}

void ClearAllSolenoids(){
  int i;

  for(i=0;i<17;i++){
    _solenoidControl[i]=0;
  }
}


void ClearAllControls(){
  PORTC &= 0xf0; //turn off the lower 4 bits.
}


void SetSolenoid(int sol, unsigned char data){
  //0=off, 1=long pulse, 2=short pulse, 3=on
  _solenoidControl[sol]=data;
}

// the loop routine runs over and over again forever:
void loop() {

  int i;

  unsigned long int count;
  unsigned char newValue;

  /* Old while(1){
   delay(100);
   }
   */

  while(1){
    //just loop through each lamp value...
    for(i=0;i<17;i++){
      //0=off, 1=long pulse, 2=short pulse, 3=on  
      switch(_solenoidControl[i]){
      case 0:
        if(i<16){
          turnOffAllSolenoids();
        }
        else{
          turnOffLED();
        }

      case 1: //long pulse

        break;

      case 2: //short pulse


        break;

      case 3: //on
        if(i<16){
          turnOnSolenoid(i);
        }
        else{
          turnOnLED();
        }

        break;
      }
    }

    //DiagLEDControl(_lightControl[22]); //calling every end of loop using lamp 22
  }	
}

ISR(TIMER1_COMPA_vect)          // timer compare interrupt service routine
{  
  if(_timerControl>3){

    _timerControl = 0;

  }



  switch(_timerControl){

  case 0:
    _slowBlink=0x00;
    _fastBlink =0x00;

    break;

  case 1:
    _slowBlink = 0x00;
    _fastBlink=0x01;

    break;

  case 2:
    _slowBlink = 0x01;
    _fastBlink = 0x00;

    break;

  case 3:
    _slowBlink = 0x01;
    _fastBlink = 0x01;

    break;

  }



  _timerControl++;

}


//2 byte format. Byte 0 = 0 off,1 = on,2 = slow,3 = fast
//Byte 1 = Which Lamp we are controlling
//Byte 1 = 64, is the LED
//65 = controlling all lamps
void receiveEvent(int howMany)
{
  int solCommand;

  digitalWrite(led, HIGH);   // turn the LED on (HIGH is the voltage level)

  solCommand = Wire.read();
  _lastSolenoid = Wire.read();

  if(solCommand < 4 && _lastSolenoid < 17)
    _solenoidControl[_lastSolenoid] = solCommand;

  if(_lastSolenoid == 16){ //since we just received a LED message..turn off the other Solenoids
    ClearAllSolenoids();
  }

  /*   Low PortC (Active High):
   0=Relay Control J3-12/J1-21 (to J?-1)
   1=Relay Control Grounds to J2-12 to 15
   2=Q18 Driver (to J?-3)
   3=Q8 Driver (to J?-4)*/
  if(solCommand==1){ //on
    switch(_lastSolenoid){
    case 17:
      PORTC |= 0x01;     //0=Relay Control J3-12/J1-21 (to J?-1) 
    case 18:
      PORTC |= 0x02;     //1=Relay Control Grounds to J2-12 to 15
    case 19:
      PORTC |= 0x04;     //2=Q18 Driver (to J?-3)
    case 20:
      PORTC |= 0x08;     //3=Q8 Driver (to J?-4)

    }
  }
  else{ //off
    switch(_lastSolenoid){
    case 17:
      PORTC &= ~0x01;     //0=Relay Control J3-12/J1-21 (to J?-1) 
    case 18:
      PORTC &= ~0x02;     //1=Relay Control Grounds to J2-12 to 15
    case 19:
      PORTC &= ~0x04;     //2=Q18 Driver (to J?-3)
    case 20:
      PORTC &= ~0x08;     //3=Q8 Driver (to J?-4) (JAF: maybe use analogWrite??)



    }
  }
}

// callback for sending last receivedValue
void sendEvent(){
  Wire.write(_lastSolenoid);
}


void turnOnSolenoid(int solID){
  unsigned char newValue;

  newValue =  PORTB | 0x0f; //reset low byte to all ####1111
  newValue = newValue & (solID | 0xf0); //example for 0xE0, this would be 0xfe AND-ing with. so new value should be ####1110
  PORTB = newValue;
}

void turnOnLED(){
  //          Serial.println ("Turning On LED");
  //          PORTB ^=(1<<PB5);
  PORTB = PORTB | 0x20; //turn on PB5 = LED 
  //          _delay_ms(500); 

}
void turnOffLED(){
  PORTB = PORTB & 0xdf; //turn off PB5 = LED   
}

void turnOffAllSolenoids(){
  PORTB |= 0x0f; //turn off.   
}





